package tokens

const (
	// The DB bucket that contains general-purpose Bazil data not tied to
	// any specific volume.
	BucketBazil = "bazil"

	// The DB bucket that contains a sub-bucket per volume, named by
	// the volume ID.
	BucketVolume = "volume"

	// The DB bucket that contains a key per volume, named by the
	// human-readable volume name.
	BucketVolName = "volname"

	// The DB bucket that contains sharing groups, for convergent
	// encryption. Key is user-friendly name, value is the 32-byte
	// secret.
	BucketSharing = "sharing"

	// The DB bucket that contains peers by public key.
	BucketPeer = "peer"

	// The DB bucket that contains peers by sequential ID. Value is
	// just the raw public key, or empty for tombstone. Peer IDs are
	// never reused.
	BucketPeerID = "peerID"
)
