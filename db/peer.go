package db

import (
	"encoding/binary"
	"errors"

	"bazil.org/bazil/kv"
	"bazil.org/bazil/kv/kvmulti"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/tokens"
	"github.com/boltdb/bolt"
)

var (
	ErrPeerNotFound      = errors.New("peer not found")
	ErrNoStorageForPeer  = errors.New("no storage offered to peer")
	ErrNoLocationForPeer = errors.New("no network location known for peer")
)

var (
	bucketPeer        = []byte(tokens.BucketPeer)
	bucketPeerID      = []byte(tokens.BucketPeerID)
	peerStateID       = []byte(tokens.PeerStateID)
	peerStateLocation = []byte(tokens.PeerStateLocation)
	peerStateStorage  = []byte(tokens.PeerStateStorage)
	peerStateVolume   = []byte(tokens.PeerStateVolume)
)

func (tx *Tx) initPeers() error {
	if _, err := tx.CreateBucketIfNotExists(bucketPeer); err != nil {
		return err
	}
	if _, err := tx.CreateBucketIfNotExists(bucketPeerID); err != nil {
		return err
	}
	return nil
}

func (tx *Tx) Peers() *Peers {
	p := &Peers{
		peers: tx.Bucket(bucketPeer),
		ids:   tx.Bucket(bucketPeerID),
	}
	return p
}

type Peers struct {
	peers *bolt.Bucket
	ids   *bolt.Bucket
}

// Get returns a Peer for the given public key.
//
// If the peer does not exist, returns ErrPeerNotFound.
func (b *Peers) Get(pub *peer.PublicKey) (*Peer, error) {
	bp := b.peers.Bucket(pub[:])
	if bp == nil {
		return nil, ErrPeerNotFound
	}
	p := &Peer{
		b:   bp,
		pub: pub,
	}
	return p, nil
}

// Make returns a Peer for the given public key, adding it if
// necessary.
func (b *Peers) Make(pub *peer.PublicKey) (*Peer, error) {
	p, err := b.Get(pub)
	if err != ErrPeerNotFound {
		// operational error or success, either is fine here
		return p, err
	}

	// really not found -> add it; first,pick a free id
	var id peer.ID
	c := b.ids.Cursor()
	if k, _ := c.Last(); k != nil {
		id = peer.ID(binary.BigEndian.Uint32(k))
	}
	id++
	if id == 0 {
		return nil, errors.New("out of peer IDs")
	}
	var idKey [4]byte
	binary.BigEndian.PutUint32(idKey[:], uint32(id))
	if err := b.ids.Put(idKey[:], pub[:]); err != nil {
		return nil, err
	}

	// create a bucket to hold information about the peer
	bp, err := b.peers.CreateBucket(pub[:])
	if err != nil {
		return nil, err
	}
	if err := bp.Put(peerStateID, idKey[:]); err != nil {
		return nil, err
	}
	if _, err := bp.CreateBucket(peerStateLocation); err != nil {
		return nil, err
	}
	if _, err := bp.CreateBucket(peerStateStorage); err != nil {
		return nil, err
	}
	if _, err := bp.CreateBucket(peerStateVolume); err != nil {
		return nil, err
	}

	p = &Peer{
		b:   bp,
		pub: pub,
	}
	return p, nil
}

func (b *Peers) Cursor() *PeersCursor {
	return &PeersCursor{b.peers.Cursor()}
}

type PeersCursor struct {
	c *bolt.Cursor
}

func (c *PeersCursor) item(k, _ []byte) *Peer {
	if k == nil {
		return nil
	}
	bucket := c.c.Bucket().Bucket(k)
	if bucket == nil {
		panic("db peer corrupt, not a bucket")
	}
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(k); err != nil {
		panic("db peer corrupt: " + err.Error())
	}
	p := &Peer{
		b:   bucket,
		pub: &pub,
	}
	return p
}

func (c *PeersCursor) First() *Peer {
	return c.item(c.c.First())
}

func (c *PeersCursor) Next() *Peer {
	return c.item(c.c.Next())
}

type Peer struct {
	b   *bolt.Bucket
	pub *peer.PublicKey
}

func (p *Peer) Pub() *peer.PublicKey {
	return p.pub
}

func (p *Peer) ID() peer.ID {
	v := p.b.Get(peerStateID)
	if v == nil {
		panic("peer corrupt, missing id: " + p.pub.String())
	}
	return peer.ID(binary.BigEndian.Uint32(v))
}

func (p *Peer) Locations() *PeerLocations {
	b := p.b.Bucket(peerStateLocation)
	return &PeerLocations{b}
}

type PeerLocations struct {
	b *bolt.Bucket
}

// Set the network location where the peer can be contacted.
//
// TODO support multiple addresses to attempt, with some idea of
// preferring recent ones.
func (p *PeerLocations) Set(addr string) error {
	// remove all the other values, for "set"
	c := p.b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		if err := p.b.Delete(k); err != nil {
			return err
		}
	}
	return p.b.Put([]byte(addr), nil)
}

// Get an address to try, to connect to the peer.
//
// Returned addr is valid after the transaction.
//
// TODO support multiple addresses to attempt, with some idea of
// preferring recent ones.
func (p *PeerLocations) Get() (addr string, err error) {
	c := p.b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		return string(k), nil
	}
	return "", ErrNoLocationForPeer
}

func (p *Peer) Storage() *PeerStorage {
	b := p.b.Bucket(peerStateStorage)
	return &PeerStorage{b}
}

type PeerStorage struct {
	b *bolt.Bucket
}

func (p *PeerStorage) Allow(backend string) error {
	return p.b.Put([]byte(backend), nil)
}

// Open key-value stores as allowed for this peer. Uses the opener
// function for the actual open action.
//
// If the peer is not allowed to use any storage, returns
// ErrNoStorageForPeer.
//
// Returned KV is valid after the transaction. Strings passed to the
// opener function are valid after the transaction.
func (p *PeerStorage) Open(opener func(string) (kv.KV, error)) (kv.KV, error) {
	var kvstores []kv.KV
	c := p.b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		backend := string(k)
		// later value may include quota style restrictions
		s, err := opener(backend)
		if err != nil {
			// TODO once kv.KV has Close, close all in kvstores
			return nil, err
		}
		kvstores = append(kvstores, s)
	}
	if len(kvstores) == 0 {
		return nil, ErrNoStorageForPeer
	}
	return kvmulti.New(kvstores...), nil
}

func (p *Peer) Volumes() *PeerVolumes {
	b := p.b.Bucket(peerStateVolume)
	return &PeerVolumes{b}
}

type PeerVolumes struct {
	b *bolt.Bucket
}

func (p *PeerVolumes) Allow(vol *Volume) error {
	return p.b.Put([]byte(vol.id), nil)
}

func (p *PeerVolumes) IsAllowed(vol *Volume) bool {
	found := p.b.Get([]byte(vol.id)) != nil
	return found
}
