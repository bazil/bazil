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

	// The DB bucket that stores alternate versions of this directory
	// entry.
	//
	// The entries may not actually be conflicting; incoming sync to
	// open files is deferred.
	//
	// Key is <dirInode:uint64_be><name>"\x00"<clock>, value is
	// protobuf bazil.snap.Dirent with fields name and clock empty.
	// For the purposes of this, the root directory has parent inode 0
	// and empty string as name.
	VolumeStateConflict = "conflict"
)
