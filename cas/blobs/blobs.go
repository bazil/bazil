package blobs

import (
	"errors"
	"fmt"
	"io"
	"math"

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
	depth uint8
}

var _ io.ReaderAt = &Blob{}
var _ io.WriterAt = &Blob{}

var ErrMissingType = errors.New("Manifest is missing Type")

// Minimum valid chunk size.
const MinChunkSize = 4096

// With min chunkSize 4kB, we might need 2**64-2**12=2**52 chunks
// (maybe -1, but that's irrelevant). With fanout>=2, that would
// mean <=52 levels, so uint8 is sufficient for level.

// SmallChunkSizeError is the error returned from Open if the
// configuration has a ChunkSize less than MinChunkSize
type SmallChunkSizeError struct {
	Given uint32
}

var _ = error(SmallChunkSizeError{})

func (s SmallChunkSizeError) Error() string {
	return fmt.Sprintf("ChunkSize is too small: %d < %d", s.Given, MinChunkSize)
}

// SmallFanoutError is the error returned from Open if the
// configuration has a Fanout less than 2.
type SmallFanoutError struct {
	Given uint32
}

var _ = error(SmallFanoutError{})

func (s SmallFanoutError) Error() string {
	return fmt.Sprintf("Fanout is too small: %d", s.Given)
}

func (blob *Blob) computeLevel(size uint64) uint8 {
	// convert size (count of bytes) to offset of last byte
	if size == 0 {
		return 0
	}

	off := size - 1
	idx := uint32(off / uint64(blob.m.ChunkSize))
	var level uint8
	for idx > 0 {
		idx /= blob.m.Fanout
		level++
	}
	return level
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
		return nil, ErrMissingType
	}
	if m.ChunkSize < MinChunkSize {
		return nil, SmallChunkSizeError{m.ChunkSize}
	}
	if m.Fanout < 2 {
		return nil, SmallFanoutError{m.Fanout}
	}
	blob := &Blob{
		stash: stash.New(chunkStore),
		m:     m,
	}
	blob.depth = blob.computeLevel(blob.m.Size)
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
	level := blob.depth

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

func (blob *Blob) grow(level uint8) error {
	// grow hash tree upward if needed

	for blob.depth < level {
		key, chunk, err := blob.stash.Clone(cas.Empty, blob.m.Type, blob.depth+1, blob.m.Fanout*cas.KeySize)
		if err != nil {
			return err
		}

		copy(chunk.Buf, blob.m.Root.Bytes())
		blob.m.Root = key
		blob.depth++
	}
	return nil
}

// chunk must be a Private chunk
func (blob *Blob) discardAfter(chunk *chunks.Chunk, lidx uint32, level uint8) error {
	if level == 0 {
		return nil
	}
	for ; lidx < blob.m.Fanout; lidx++ {
		keyoff := lidx * cas.KeySize
		keybuf := chunk.Buf[keyoff : keyoff+cas.KeySize]
		key := cas.NewKeyPrivate(keybuf)
		if key.IsPrivate() {
			// there can't be any Private chunks if they key wasn't Private
			chunk, err := blob.stash.Get(key, blob.m.Type, level-1)
			if err != nil {
				return err
			}
			err = blob.discardAfter(chunk, 0, level-1)
			if err != nil {
				return err
			}
			blob.stash.Drop(key)
		}
		copy(chunk.Buf[keyoff:keyoff+cas.KeySize], cas.Empty.Bytes())
	}
	return nil
}

// Decreases depth, always selecting only the leftmost tree,
// and dropping all Private chunks in the rest.
func (blob *Blob) shrink(level uint8) error {
	for blob.depth > level {
		chunk, err := blob.stash.Get(blob.m.Root, blob.m.Type, blob.depth)
		if err != nil {
			return err
		}

		if blob.m.Root.IsPrivate() {
			// blob.depth must be >0 if we're here, so it's always a
			// pointer chunk; iterate all non-first keys and drop
			// Private chunks
			err = blob.discardAfter(chunk, 1, blob.depth)
			if err != nil {
				return err
			}
		}

		// now all non-left top-level private nodes have been dropped
		keybuf := safeSlice(chunk.Buf, 0, cas.KeySize)
		key := cas.NewKeyPrivate(keybuf)
		blob.m.Root = key
		blob.depth--
	}
	return nil
}

// lookupForWrite fetches the data chunk for the given offset and
// ensures it is Private and reinflated, and thus writable.
func (blob *Blob) lookupForWrite(off uint64) (*chunks.Chunk, error) {
	gidx := uint32(off / uint64(blob.m.ChunkSize))
	lidxs := localChunkIndexes(blob.m.Fanout, gidx)

	err := blob.grow(uint8(len(lidxs)))
	if err != nil {
		return nil, err
	}

	level := blob.depth

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

func zeroSlice(p []byte) {
	for len(p) > 0 {
		p[0] = 0
		p = p[1:]
	}
}

const debugTruncate = true

// Truncate adjusts the size of the blob. If the new size is less than
// the old size, data past that point is lost. If the new size is
// greater than the old size, the new part is full of zeroes.
func (blob *Blob) Truncate(size uint64) error {
	switch {
	case size == 0:
		// special case shrink to nothing
		blob.m.Root = cas.Empty
		blob.m.Size = 0
		blob.stash.Clear()

	case size < blob.m.Size:
		// shrink

		// i really am starting to hate the idea of file offsets being
		// int64's, but can't fight all the windmills at once.
		if size > math.MaxInt64 {
			return errors.New("cannot discard past 63-bit file size")
		}

		// we know size>0 from above
		off := size - 1
		gidx := uint32(off / uint64(blob.m.ChunkSize))
		lidxs := localChunkIndexes(blob.m.Fanout, gidx)
		err := blob.shrink(uint8(len(lidxs)))
		if err != nil {
			return err
		}

		// we don't need to always cow here (if everything is
		// perfectly aligned / already zero), but it's a rare enough
		// case that let's not care for now
		//
		// TODO this makes a tight loop on Open and Save wasteful

		{
			// TODO clone all the way down to be able to trim leaf chunk,
			// abusing lookupForWrite for now

			// we know size > 0 from above
			_, err := blob.lookupForWrite(size - 1)
			if err != nil {
				return err
			}
		}

		// now zero-fill on the right; guaranteed cow by the above kludge
		key := blob.m.Root
		if debugTruncate {
			if !key.IsPrivate() {
				panic(fmt.Errorf("Truncate root is not private: %v", key))
			}
		}
		for level := blob.depth; level > 0; level-- {
			chunk, err := blob.stash.Get(key, blob.m.Type, level)
			if err != nil {
				return err
			}
			err = blob.discardAfter(chunk, lidxs[level-1]+1, level)
			if err != nil {
				return err
			}
			keyoff := int64(lidxs[level-1]) * cas.KeySize
			keybuf := chunk.Buf[keyoff : keyoff+cas.KeySize]
			key = cas.NewKeyPrivate(keybuf)
			if debugTruncate {
				if !key.IsPrivate() {
					panic(fmt.Errorf("Truncate key at level %d not private: %v", level, key))
				}
			}
		}

		// and finally the leaf chunk
		chunk, err := blob.stash.Get(key, blob.m.Type, 0)
		if err != nil {
			return err
		}
		{
			// TODO is there anything to clear here; beware modulo wraparound

			// size is also the offset of the next byte
			loff := uint32(size % uint64(blob.m.ChunkSize))
			zeroSlice(chunk.Buf[loff:])
		}

		// TODO what's the right time to adjust size, wrt errors
		blob.m.Size = size

		// TODO unit tests that checks we don't leak chunks?

	case size > blob.m.Size:
		// grow
		off := size - 1
		gidx := uint32(off / uint64(blob.m.ChunkSize))
		lidxs := localChunkIndexes(blob.m.Fanout, gidx)
		err := blob.grow(uint8(len(lidxs)))
		if err != nil {
			return err
		}
		blob.m.Size = size
	}
	return nil
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
	// make sure the tree is optimal depth, as later we rely purely on
	// size to compute depth; this might happen because of errors on a
	// write/truncate path
	level := blob.computeLevel(blob.m.Size)
	switch {
	case blob.depth > level:
		err := blob.shrink(level)
		if err != nil {
			return nil, err
		}
	case blob.depth < level:
		err := blob.grow(level)
		if err != nil {
			return nil, err
		}
	}
	k, err := blob.saveChunk(blob.m.Root, blob.depth)
	if err != nil {
		return nil, err
	}
	blob.m.Root = k
	// make a copy to return
	m := blob.m
	return &m, nil
}
