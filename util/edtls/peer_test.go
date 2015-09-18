package edtls_test

import (
	"crypto/rand"
	"crypto/tls"
	"io/ioutil"
	"net"
	"sync"
	"testing"

	"bazil.org/bazil/util/edtls"
	"github.com/agl/ed25519"
)

func TestPeer(t *testing.T) {
	clientPublicKey, clientPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientConfig := mustGenerateTLSConfig(t, clientPublicKey, clientPrivateKey)
	clientConfig.InsecureSkipVerify = true

	serverPublicKey, serverPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serverConfig := mustGenerateTLSConfig(t, serverPublicKey, serverPrivateKey)

	pipeClient, pipeServer := net.Pipe()

	var seen []byte
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer t.Logf("client done")
		defer pipeClient.Close()
		t.Logf("client new")
		conn, err := edtls.NewClient(pipeClient, clientConfig, serverPublicKey)
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("client writing")
		if _, err := conn.Write([]byte("Greetings")); err != nil {
			t.Error(err)
			return
		}
		t.Logf("client closing")
		if err := conn.Close(); err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer t.Logf("server done")
		defer pipeServer.Close()
		t.Logf("server new")
		conn := tls.Server(pipeServer, serverConfig)
		if err != nil {
			t.Error(err)
			return
		}
		if err := conn.Handshake(); err != nil {
			conn.Close()
			t.Error(err)
			return
		}
		state := conn.ConnectionState()
		if !state.HandshakeComplete {
			t.Error("TLS handshake did not complete")
			return
		}
		if len(state.PeerCertificates) == 0 {
			t.Error("no TLS peer certificates")
			return
		}
		t.Logf("server verifying")
		remotePublicKey, ok := edtls.Verify(state.PeerCertificates[0])
		if !ok {
			t.Error("edtls verification failed")
			return
		}
		if *remotePublicKey != *clientPublicKey {
			t.Errorf("wrong client public key: %x != %x", *remotePublicKey, *clientPublicKey)
			return
		}
		t.Logf("server reading")
		buf, err := ioutil.ReadAll(conn)
		if err != nil {
			t.Error(err)
			return
		}
		seen = buf
	}()

	wg.Wait()

	if seen == nil {
		t.Fatalf("did not pass greeting")
	}
	if g, e := string(seen), "Greetings"; g != e {
		t.Fatalf("greeting does not match: %q != %q", g, e)
	}
}
