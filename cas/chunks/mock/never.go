package mock

import (
	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
)

// NeverUsed is a chunks.Store meant for unit tests that don't touch
// the CAS, but where the API requires one.
type NeverUsed struct{}

var _ = chunks.Store(NeverUsed{})

// Get fetches a Chunk. See chunks.Store.Get.
func (NeverUsed) Get(key cas.Key, typ string, level uint8) (*chunks.Chunk, error) {
	panic("NeverUsed.Get was called")
}

// Add adds a Chunk to the Store. See chunks.Store.Add.
func (NeverUsed) Add(chunk *chunks.Chunk) (key cas.Key, err error) {
	panic("NeverUsed.Add was called")
}
