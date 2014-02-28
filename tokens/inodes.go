package tokens

// the top half of inode space is reserved for dynamically allocated
// inodes
const (
	InodeKindMask    uint64 = 1 << 63
	InodeKindDynamic uint64 = 1 << 63
	InodeKindNormal  uint64 = 0
)
