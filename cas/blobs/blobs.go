package blobs

import (
	"errors"
	"fmt"
	"io"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/stash"
)

const debugLookup = true

// Implementation terminology:
//
// Bytes have offsets, and chunks have indexes.
//
// Both came in global and local flavors.
//
// Byte local offsets are within a chunk.
//
// Chunk local indexes are within a level. There are two kinds of
// chunks: data chunks and pointer chunks. Only data chunks have
// global indexes, while both kind have local indexes.
//
// Example: global byte offset at 5MB might be 1MB into the second
// chunk.
//
// Example: The chunk with global index 70 might have local indexes 1
// (for level 1) and 6 (for level 0), if each pointer chunk held 64
// pointers.

// Manifest is a description of a Blob as persisted in a chunks.Store.
//
// When creating a new Blob, create a Manifest and set the Type,
// ChunkSize and Fanout fields, the rest can be left to their zero
// values. See EmptyManifest for a helper that uses default tuning.
type Manifest struct {
	Type string
	Root cas.Key
	Size uint64
	// Must be >= MinChunkSize.
	ChunkSize uint32
	// Must be >= 2.
	Fanout uint32
}

// EmptyManifest returns an empty manifest of the given type with the
// default tuning parameters.
func EmptyManifest(type_ string) *Manifest {
	const kB = 1024
	const MB = 1024 * kB

	return &Manifest{
		Type:      type_,
		ChunkSize: 4 * MB,
		Fanout:    64,
	}
}

// Blob is a container for arbitrary size data (byte sequence),
// constructed from lower-level Chunks.
type Blob struct {
	stash *stash.Stash
	m     Manifest
}

var _ io.ReaderAt = &Blob{}
var _ io.WriterAt = &Blob{}

var MissingType = errors.New("Manifest is missing Type")

// Minimum valid chunk size.
const MinChunkSize = 4096

// With min chunkSize 4kB, we might need 2**64-2**12=2**52 chunks
// (maybe -1, but that's irrelevant). With fanout>=2, that would
// mean <=52 levels, so uint8 is sufficient for level.

// SmallChunkSize is the error returned from Open if the configuration
// has a ChunkSize less than MinChunkSize
type SmallChunkSize struct {
	Given uint32
}

var _ = error(SmallChunkSize{})

func (s SmallChunkSize) Error() string {
	return fmt.Sprintf("ChunkSize is too small: %d < %d", s.Given, MinChunkSize)
}

// SmallFanout is the error returned from Open if the configuration
// has a Fanout less than 2.
type SmallFanout struct {
	Given uint32
}

var _ = error(SmallFanout{})

func (s SmallFanout) Error() string {
	return fmt.Sprintf("Fanout is too small: %d", s.Given)
}

// Open returns a new Blob, using the given chunk store and manifest.
//
// It makes a copy of the manifest, so the caller is free to use it in
// any way after the call.
//
// A Blob need not exist; passing in a Manifest with an Empty Root
// gives a Blob with zero contents. However, all the fields must be
// set to valid values.
func Open(chunkStore chunks.Store, manifest *Manifest) (*Blob, error) {
	// make a copy so caller can't mutate it
	m := *manifest
	if m.Type == "" {
		return nil, MissingType
	}
	if m.ChunkSize < MinChunkSize {
		return nil, SmallChunkSize{m.ChunkSize}
	}
	if m.Fanout < 2 {
		return nil, SmallFanout{m.Fanout}
	}
	blob := &Blob{
		stash: stash.New(chunkStore),
		m:     m,
	}
	return blob, nil
}

func (blob *Blob) String() string {
	return fmt.Sprintf("Blob{type:%q root:%v size:%d}",
		blob.m.Type, blob.m.Root, blob.m.Size)
}

// Given a global chunk index, generate a list of local chunk indexes.
//
// The list needs to be generated bottom up, but we consume it top
// down, so generate it fully at the beginning and keep it as a slice.
func localChunkIndexes(fanout uint32, chunk uint32) []uint32 {
	// 6 is a good guess for max level of pointer chunks;
	// 4MiB chunksize, uint32 chunk index -> 15PiB of data.
	// overflow just means an allocation.
	index := make([]uint32, 0, 6)

	for chunk > 0 {
		index = append(index, chunk%fanout)
		chunk /= fanout
	}
	return index
}

// safeSlice returns a slice of buf if possible, and where buf is not
// large enough to serve this slice, it returns a new slice of the
// right size. In case buf ends in the middle of the range, the
// available bytes are copied over to the new slice.
func safeSlice(buf []byte, low int, high int) []byte {
	if high <= len(buf) {
		return buf[low:high]
	}
	s := make([]byte, high-low)
	if low <= len(buf) {
		copy(s, buf[low:])
	}
	return s
}

// lookup fetches the data chunk for given global byte offset.
//
// The returned Chunk remains zero trimmed.
//
// It may be a Private or a normal chunk. For writable Chunks, call
// lookupForWrite instead.
func (blob *Blob) lookup(off uint64) (*chunks.Chunk, error) {
	gidx := uint32(off / uint64(blob.m.ChunkSize))
	lidxs := localChunkIndexes(blob.m.Fanout, gidx)
	level := blob.level()

	// walk down from the root
	var ptrKey = blob.m.Root
	for ; level > 0; level-- {
		// follow pointer chunks
		var idx uint32
		if int(level)-1 < len(lidxs) {
			idx = lidxs[level-1]
		}

		chunk, err := blob.stash.Get(ptrKey, blob.m.Type, level)
		if err != nil {
			return nil, err
		}

		keyoff := int64(idx) * cas.KeySize
		// zero trimming may have cut the key off, even in the middle
		// TODO ugly int conversion
		keybuf := safeSlice(chunk.Buf, int(keyoff), int(keyoff+cas.KeySize))
		ptrKey = cas.NewKeyPrivate(keybuf)
	}

	chunk, err := blob.stash.Get(ptrKey, blob.m.Type, 0)
	return chunk, err
}

// ReadAt reads data from the given offset. See io.ReaderAt.
func (blob *Blob) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("negative offset is not possible")
	}
	{
		off := uint64(off)
		for {
			if off >= blob.m.Size {
				return n, io.EOF
			}

			// avoid reading past EOF
			if uint64(len(p)) > blob.m.Size-off {
				p = p[:int(blob.m.Size-off)]
			}

			if len(p) == 0 {
				break
			}

			chunk, err := blob.lookup(off)
			if err != nil {
				return n, err
			}

			loff := uint32(off % uint64(blob.m.ChunkSize))
			var copied int
			// TODO ugly int conversion
			if int(loff) <= len(chunk.Buf) {
				copied = copy(p, chunk.Buf[loff:])
			}
			for len(p) > copied && loff+uint32(copied) < blob.m.ChunkSize {
				// handle case where chunk has been zero trimmed
				p[copied] = '\x00'
				copied++
			}
			n += copied
			p = p[copied:]
			off += uint64(copied)
		}

	}
	return n, err
}

func (blob *Blob) chunkSizeForLevel(level uint8) uint32 {
	switch level {
	case 0:
		return blob.m.ChunkSize
	default:
		return blob.m.Fanout * cas.KeySize
	}
}

func (blob *Blob) level() uint8 {
	// convert size (count of bytes) to offset of last byte
	if blob.m.Size == 0 {
		return 0
	}
	off := blob.m.Size - 1

	idx := uint32(off / uint64(blob.m.ChunkSize))
	var level uint8
	for idx > 0 {
		idx /= blob.m.Fanout
		level++
	}
	return level
}

// lookupForWrite fetches the data chunk for the given offset and
// ensures it is Private and reinflated, and thus writable.
func (blob *Blob) lookupForWrite(off uint64) (*chunks.Chunk, error) {
	gidx := uint32(off / uint64(blob.m.ChunkSize))
	lidxs := localChunkIndexes(blob.m.Fanout, gidx)
	level := blob.level()

	// grow hash tree upward if needed
	for int(level) < len(lidxs) {
		key, chunk, err := blob.stash.Clone(cas.Empty, blob.m.Type, level+1, blob.m.Fanout*cas.KeySize)
		if err != nil {
			return nil, err
		}

		copy(chunk.Buf, blob.m.Root.Bytes())
		// TODO don't change Root until you change Size, or errors lead to corruption
		blob.m.Root = key
		level += 1
	}

	var parentChunk *chunks.Chunk
	{
		// clone root if necessary
		var k cas.Key
		var err error
		size := blob.chunkSizeForLevel(level)
		k, parentChunk, err = blob.stash.Clone(blob.m.Root, blob.m.Type, level, size)
		if err != nil {
			return nil, err
		}
		blob.m.Root = k
	}

	// walk down from the root
	var ptrKey = blob.m.Root
	for ; level > 0; level-- {
		// follow pointer chunks
		var idx uint32
		if int(level)-1 < len(lidxs) {
			idx = lidxs[level-1]
		}

		keyoff := int64(idx) * cas.KeySize
		{
			k := cas.NewKeyPrivate(parentChunk.Buf[keyoff : keyoff+cas.KeySize])
			if k.IsReserved() {
				return nil, fmt.Errorf("invalid stored key: key @%d in %v is %v", keyoff, ptrKey, parentChunk.Buf[keyoff:keyoff+cas.KeySize])
			}
			ptrKey = k
		}

		// clone it (nop if already cloned)
		size := blob.chunkSizeForLevel(level - 1)
		ptrKey, child, err := blob.stash.Clone(ptrKey, blob.m.Type, level-1, size)
		if err != nil {
			return nil, err
		}

		if debugLookup {
			if uint64(len(child.Buf)) != uint64(size) {
				panic(fmt.Errorf("lookupForWrite clone for level %d made weird size %d != %d, key %v", level-1, len(child.Buf), size, ptrKey))
			}
		}

		// update the key in parent
		n := copy(parentChunk.Buf[keyoff:keyoff+cas.KeySize], ptrKey.Bytes())
		if debugLookup {
			if n != cas.KeySize {
				panic(fmt.Errorf("lookupForWrite copied only %d of the key", n))
			}
		}
		parentChunk = child
	}

	if debugLookup {
		if parentChunk.Level != 0 {
			panic(fmt.Errorf("lookupForWrite got a non-leaf: %v", parentChunk.Level))
		}
		if uint64(len(parentChunk.Buf)) != uint64(blob.m.ChunkSize) {
			panic(fmt.Errorf("lookupForWrite got short leaf: %v", len(parentChunk.Buf)))
		}
	}

	return parentChunk, nil
}

// WriteAt writes data to the given offset. See io.WriterAt.
func (blob *Blob) WriteAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("negative offset is not possible")
	}
	{
		off := uint64(off)
		for len(p) > 0 {

			chunk, err := blob.lookupForWrite(off)
			if err != nil {
				return n, err
			}

			loff := uint32(off % uint64(blob.m.ChunkSize))
			copied := copy(chunk.Buf[loff:], p)
			n += copied
			p = p[copied:]
			off += uint64(copied)

			// off points now at the *next* byte that would be
			// written, so the "byte offset 0 is size 1" logic works
			// out here without -1's
			if off > blob.m.Size {
				blob.m.Size = off
			}
		}
	}
	return n, nil
}

// Size returns the current byte size of the Blob.
func (blob *Blob) Size() uint64 {
	return blob.m.Size
}

func trim(b []byte) []byte {
	end := len(b)
	for end > 0 && b[end-1] == 0x00 {
		end--
	}
	return b[:end]
}

func (blob *Blob) saveChunk(key cas.Key, level uint8) (cas.Key, error) {
	if !key.IsPrivate() {
		// already saved
		return key, nil
	}

	chunk, err := blob.stash.Get(key, blob.m.Type, level)
	if err != nil {
		return key, err
	}

	if level > 0 {
		for off := uint32(0); off+cas.KeySize <= uint32(len(chunk.Buf)); off += cas.KeySize {
			cur := cas.NewKeyPrivate(chunk.Buf[off : off+cas.KeySize])
			if cur.IsReserved() {
				return key, fmt.Errorf("invalid stored key: key @%d in %v is %v", off, key, chunk.Buf[off:off+cas.KeySize])
			}
			// recurses at most `level` deep
			saved, err := blob.saveChunk(cur, level-1)
			if err != nil {
				return key, err
			}
			copy(chunk.Buf[off:off+cas.KeySize], saved.Bytes())
		}
	}

	chunk.Buf = trim(chunk.Buf)
	return blob.stash.Save(key)
}

// Save persists the Blob into the Store and returns a new Manifest
// that can be passed to Open later.
func (blob *Blob) Save() (*Manifest, error) {
	k, err := blob.saveChunk(blob.m.Root, blob.level())
	if err != nil {
		return nil, err
	}
	blob.m.Root = k
	// make a copy to return
	m := blob.m
	return &m, nil
}
