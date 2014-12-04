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
package edtls
