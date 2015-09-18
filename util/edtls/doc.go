// Package edtls provides ed25519 signatures on top of TLS certificates.
//
// There is currently no standard way to use ed25519 or curve25519
// cryptographic algorithms in TLS. See drafts at
// http://ietfreport.isoc.org/idref/draft-josefsson-tls-curve25519/
// and http://ietfreport.isoc.org/idref/draft-josefsson-eddsa-ed25519/
// for standardization attempts.
//
// The way the TLS protocol is designed, it relies on centralized
// registries of algorithms. We cannot easily plug in a new kind of a
// certificate. Instead, we abuse the extension mechanism to transmit
// an extra, custom, certificate.
//
// Client connecting to servers are expected to already know the
// ed25519 public key of the server. Clients will announce their
// public key, and the server-side logic can use that for
// authentication and access control.
//
// In both directions a "vouch" is transmitted as a TLS extension. It
// contains an ed25519 public key and a signature of the certificate
// expiry time and the DER-encoded TLS public key.
//
// If a vouch packet opens without errors, and contents match the TLS
// public key of the sender, the receiver knows that the sender
// actually owns the ed25519 public key and the TLS public key.
//
// Vouches cryptographically verify the expiry time of the TLS
// certificate, to make sure that an attacker did not manage to just
// steal the TLS private key, but also holds the ed25519 private key.
// As the TLS private key lives in the same memory space as the
// ed25519 private keys, an attack may be able to steal both, but
// off-the-shelf attacks will typically only target the TLS key.
//
// There is currently no mechanism to rotate the ed25519 keys.
package edtls
