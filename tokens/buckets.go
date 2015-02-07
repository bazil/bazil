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
)
