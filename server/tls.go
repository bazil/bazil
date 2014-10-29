package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"time"
)

const (
	tlsExpiry = 4 * time.Hour
	// generate a new cert when it has less than this much time left
	tlsRegen = 1 * time.Hour
)

func (*App) generateTLSConfig() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// generate a self-signed cert
	now := time.Now()
	expiry := now.Add(tlsExpiry)
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
	certDER, err := x509.CreateCertificate(rand.Reader, srvTemplate, srvTemplate, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	// parse it back because CreateCertificate API seems to insist on
	// just giving us the DER form
	x509Cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	var cert = tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
		// populated primarily to neatly expose NotAfter to
		// getTLSConfig, but might help performance too
		Leaf: x509Cert,
	}

	var conf = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      x509.NewCertPool(),
		ClientAuth:   tls.RequestClientCert,
		MinVersion:   tls.VersionTLS12,
	}

	return conf, nil
}

func (app *App) GetTLSConfig() (*tls.Config, error) {
	v := app.tls.config.Load()
	if v != nil {
		conf := v.(*tls.Config)
		now := time.Now()
		expires := conf.Certificates[0].Leaf.NotAfter
		if expires.After(now.Add(tlsRegen)) {
			// all good
			return conf, nil
		}
	}

	// haven't generated a cert yet, or it's about to expire
	app.tls.gen.Lock()
	defer app.tls.gen.Unlock()
	if v2 := app.tls.config.Load(); v2 != v {
		// lost the race, someone else generated it already
		conf := v2.(*tls.Config)
		return conf, nil
	}
	// we now hold the lock and really should generate the cert; error
	// here just means others will try again
	conf, err := app.generateTLSConfig()
	if err == nil {
		app.tls.config.Store(conf)
	}
	return conf, err
}
