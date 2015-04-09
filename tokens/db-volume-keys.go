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

	// The DB bucket that stores logical clocks tracking file
	// changes.
	//
	// Key is <dirInode:uint64_be><name>, value is binary marshaled
	// clock.Version. For the purposes of this, the root directory has
	// parent inode 0 and empty string as name.
	VolumeStateClock = "clock"
)
