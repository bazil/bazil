package chunkutil

import (
	"fmt"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"github.com/codahale/blake2"
)

const personalizationPrefix = "bazil:"

// To guarantee hash output never matches reservedPrefix, matching
// output is overwritten with this prefix.
//
// C0111DED = COLLIDED
var replaceSpecial = []byte{0xC0, 0x11, 0x1D, 0xED, 0x00}

// Hash hashes the data in a chunk into a cas.Key.
//
// Hash makes sure to never return a Special Key.
func Hash(chunk *chunks.Chunk) cas.Key {
	var pers [blake2.PersonalSize]byte
	copy(pers[:], personalizationPrefix)
	copy(pers[len(personalizationPrefix):], chunk.Type)
	config := &blake2.Config{
		Size:     cas.KeySize,
		Personal: pers[:],
		Tree: &blake2.Tree{
			// We are faking tree mode without any intent to actually
			// follow all the rules, to be able to feed the level
			// into the hash function. These settings are dubious, but
			// we need to do something to make having Tree legal.
			Fanout:        0,
			MaxDepth:      255,
			InnerHashSize: cas.KeySize,

			NodeDepth: chunk.Level,
		},
	}
	h := blake2.New(config)
	if len(chunk.Buf) == 0 {
		return cas.Empty
	}
	_, _ = h.Write(chunk.Buf)
	keybuf := h.Sum(nil)
	return makeKey(keybuf)
}

func makeKey(keybuf []byte) cas.Key {
	key := cas.NewKey(keybuf)
	if key.IsSpecial() {
		// Tough luck, we happened to hash one of the reserved byte
		// sequences. You should play lotto today.
		copy(keybuf, replaceSpecial)
		key = cas.NewKey(keybuf)

		if key.IsSpecial() {
			// This has to mean replaceSpecial was poorly chosen, and
			// does not avoid the special prefixes.
			panic(fmt.Errorf("replaceSpecial is still Special: %x", keybuf))
		}
	}
	return key
}
