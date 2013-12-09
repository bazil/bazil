package chunkutil

import (
	"fmt"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"github.com/dchest/blake2b"
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
	var pers [blake2b.PersonSize]byte
	copy(pers[:], personalizationPrefix)
	copy(pers[len(personalizationPrefix):], chunk.Type)
	config := &blake2b.Config{
		Size:   cas.KeySize,
		Person: pers[:],
		Tree: &blake2b.Tree{
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
	h, err := blake2b.New(config)
	if err != nil {
		// we don't let outside data directly influence the config, so
		// this is always localized programmer error
		panic(fmt.Errorf("blake2b config error: %v", err))
	}

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
