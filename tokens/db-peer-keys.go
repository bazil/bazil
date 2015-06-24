package tokens

// Keys in the bucket BucketPeer/PUB
const (
	PeerStateID = "id"

	// The DB bucket that contains addresses for the peer. Key is peer
	// host:port, value is empty for now
	PeerStateLocation = "location"

	// The DB bucket that configures what storage to offer to peer.
	// Key is storage backend, value is empty for now. Later this may
	// include quota style restrictions.
	PeerStateStorage = "storage"

	// The DB bucket that configures what volumes peer can see.
	// Key is volume ID, value is empty for now.
	PeerStateVolume = "volume"
)
