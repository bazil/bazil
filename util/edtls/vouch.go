package edtls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"

	"github.com/agl/ed25519"
)

// generated with a reimplementation of
// https://gallery.technet.microsoft.com/scriptcenter/56b78004-40d0-41cf-b95e-6e795b2e8a06
// via http://msdn.microsoft.com/en-us/library/ms677620(VS.85).aspx
var oid = asn1.ObjectIdentifier{1, 2, 840, 113556, 1, 8000, 2554, 31830, 5190, 18203, 20240, 41147, 7688498, 2373901}

const prefix = "vouch-tls\n"

// Vouch a self-signed certificate that is about to be created with an Ed25519 signature.
func Vouch(signPub *[ed25519.PublicKeySize]byte, signPriv *[ed25519.PrivateKeySize]byte, cert *x509.Certificate, tlsPub interface{}) error {
	// note: this is so early the cert is not serialized yet, can't use those fields
	tlsPubDer, err := x509.MarshalPKIXPublicKey(tlsPub)
	if err != nil {
		return err
	}
	msg := make([]byte, 0, len(prefix)+8+len(tlsPubDer))
	msg = append(msg, prefix...)
	var now [8]byte
	binary.LittleEndian.PutUint64(now[:], uint64(cert.NotAfter.Unix()))
	msg = append(msg, now[:]...)
	msg = append(msg, tlsPubDer...)

	env := make([]byte, 0, ed25519.PublicKeySize+ed25519.SignatureSize)
	env = append(env, signPub[:]...)
	sig := ed25519.Sign(signPriv, msg)
	env = append(env, sig[:]...)
	ext := pkix.Extension{Id: oid, Value: env}
	cert.ExtraExtensions = append(cert.ExtraExtensions, ext)
	return nil
}

func findSig(cert *x509.Certificate, pub *[ed25519.PublicKeySize]byte, sig *[ed25519.SignatureSize]byte) bool {
	for _, ext := range cert.Extensions {
		if !ext.Id.Equal(oid) {
			continue
		}
		if len(ext.Value) != ed25519.PublicKeySize+ed25519.SignatureSize {
			continue
		}
		copy(pub[:], ext.Value)
		copy(sig[:], ext.Value[ed25519.PublicKeySize:])
		return true
	}
	return false
}

// Verify a vouch as offered by the TLS peer.
//
// Returns the signing public key. It is up to the caller to decide
// whether this key is acceptable.
//
// Does not verify cert.NotAfter against a clock, just its
// authenticity.
func Verify(cert *x509.Certificate) (*[ed25519.PublicKeySize]byte, bool) {
	var pub [ed25519.PublicKeySize]byte
	var sig [ed25519.SignatureSize]byte
	if !findSig(cert, &pub, &sig) {
		return nil, false
	}

	tlsPubDer, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, false
	}
	msg := make([]byte, 0, len(prefix)+8+len(tlsPubDer))
	msg = append(msg, prefix...)
	var now [8]byte
	binary.LittleEndian.PutUint64(now[:], uint64(cert.NotAfter.Unix()))
	msg = append(msg, now[:]...)
	msg = append(msg, tlsPubDer...)

	if !ed25519.Verify(&pub, msg, &sig) {
		return nil, false
	}
	return &pub, true
}
