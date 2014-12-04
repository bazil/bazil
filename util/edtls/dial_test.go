package edtls_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"io"
	"io/ioutil"
	"math/big"
	"sync"
	"testing"
	"time"

	"bazil.org/bazil/util/edtls"
	"github.com/agl/ed25519"
)

var (
	testKeyPub  *[ed25519.PublicKeySize]byte
	testKeyPriv *[ed25519.PrivateKeySize]byte
)

func init() {
	var err error
	testKeyPub, testKeyPriv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
}

func mustGenerateTLSConfig(t *testing.T, signPub *[ed25519.PublicKeySize]byte, signPriv *[ed25519.PrivateKeySize]byte) *tls.Config {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// generate a self-signed cert
	now := time.Now()
	expiry := now.Add(1 * time.Hour)
	srvKeyID := sha1.Sum(key.D.Bytes())
	hostname := hex.EncodeToString(srvKeyID[:]) + ".peer.bazil.org"
	srvTemplate := &x509.Certificate{
		SerialNumber: new(big.Int),
		Subject: pkix.Name{
			CommonName:   hostname,
			Organization: []string{"bazil.org#peer"},
		},
		NotBefore: now.UTC().AddDate(0, 0, -7),
		NotAfter:  expiry.UTC(),

		SubjectKeyId: srvKeyID[:],
		DNSNames:     []string{hostname},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
	}

	if signPub != nil {
		if err := edtls.Vouch(signPub, signPriv, srvTemplate, &key.PublicKey); err != nil {
			t.Fatal(err)
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, srvTemplate, srvTemplate, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	var cert = tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	var conf = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      x509.NewCertPool(),
		ClientAuth:   tls.RequestClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	return conf
}

// TODO this code is ugly
// TODO test coverage for error cases
func TestDialNotEd(t *testing.T) {
	confSrv := mustGenerateTLSConfig(t, nil, nil)
	l, err := tls.Listen("tcp", "localhost:0", confSrv)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	quit := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			c, err := l.Accept()
			select {
			case <-quit:
				// nevermind the error, it's probably unexported
				// errClosing anyway
				return
			default:
			}
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = io.Copy(ioutil.Discard, c)
			c.Close()
		}
	}()

	confClient := mustGenerateTLSConfig(t, nil, nil)
	confClient.InsecureSkipVerify = true
	c, err := edtls.Dial(nil, "tcp", l.Addr().String(), confClient, testKeyPub)
	if err == nil {
		c.Close()
		t.Fatal("expected an error")
	}
	if err != edtls.ErrNotEdTLS {
		t.Fatalf("expected ErrNotEdTLS, got %T: %v", err, err)
	}

	close(quit)
	l.Close()
	wg.Wait()
}
