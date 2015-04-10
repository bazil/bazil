package tokens

// Keys in the bucket BucketVolume/VOLID
const (
	VolumeStateDir   = "dir"
	VolumeStateInode = "inode"
	VolumeStateSnap  = "snap"
	VolumeStateEpoch = "epoch"

	// The DB bucket that configures what storage the volume uses.
	// Key is name, value is protobuf bazil.db.VolumeStorage.
	VolumeStateStorage = "storage"
)
