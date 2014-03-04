package pb

// MarshalerTo describes the MarshalTo method for protobuf
// marshaling.
//
// It is provided here because gogoprotobuf does not provide a public
// definition.
type Marshaler interface {
	Size() int
	MarshalTo(data []byte) (n int, err error)
}
