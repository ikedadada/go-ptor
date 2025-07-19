package entity

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net"
	"sync"

	"ikedadada/go-ptor/internal/domain/value_object"
)

// ---- StreamState ----------------------------------------------------------

type StreamState struct {
	ID     value_object.StreamID
	Closed bool
	// 追加情報が欲しければここに (bytesSent/recv など)
}

// ---- Circuit --------------------------------------------------------------

type Circuit struct {
	id   value_object.CircuitID
	hops []value_object.RelayID

	keys          map[int]value_object.AESKey // per-hop AES key
	baseNonces    map[int]value_object.Nonce  // per-hop base Nonce
	beginCounter  map[int]uint64              // per-hop BEGIN counter
	dataCounter   map[int]uint64              // per-hop DATA counter (downstream)
	upstreamDataCounter map[int]uint64        // per-hop upstream DATA counter
	priv        *rsa.PrivateKey
	conns       []net.Conn
	strmMu      sync.RWMutex
	stream      map[value_object.StreamID]*StreamState
}

// NewCircuit は 3 ホップ分の RelayID と鍵束を受け取って生成。
func NewCircuit(id value_object.CircuitID, relays []value_object.RelayID,
	keys []value_object.AESKey, nonces []value_object.Nonce, priv *rsa.PrivateKey) (*Circuit, error) {

	if len(relays) == 0 || len(relays) != len(keys) || len(keys) != len(nonces) {
		return nil, errors.New("hops / keys / nonces length mismatch")
	}
	if priv == nil {
		return nil, errors.New("rsa key required")
	}
	keyMap := make(map[int]value_object.AESKey, len(keys))
	ncMap := make(map[int]value_object.Nonce, len(nonces))
	beginCounterMap := make(map[int]uint64, len(nonces))
	dataCounterMap := make(map[int]uint64, len(nonces))
	upstreamDataCounterMap := make(map[int]uint64, len(nonces))
	for i := range keys {
		keyMap[i] = keys[i]
		ncMap[i] = nonces[i]
		beginCounterMap[i] = 0
		dataCounterMap[i] = 0
		upstreamDataCounterMap[i] = 0
	}
	return &Circuit{
		id:           id,
		hops:         relays,
		keys:         keyMap,
		baseNonces:   ncMap,
		beginCounter: beginCounterMap,
		dataCounter:  dataCounterMap,
		upstreamDataCounter: upstreamDataCounterMap,
		priv:         priv,
		conns:        make([]net.Conn, len(relays)),
		stream:       make(map[value_object.StreamID]*StreamState),
	}, nil
}

// ----------------------------------------------------------------------------
// 不変部

func (c *Circuit) ID() value_object.CircuitID { return c.id }
func (c *Circuit) Hops() []value_object.RelayID {
	return append([]value_object.RelayID(nil), c.hops...)
}
func (c *Circuit) HopKey(idx int) value_object.AESKey  { return c.keys[idx] }
func (c *Circuit) HopBaseNonce(idx int) value_object.Nonce { return c.baseNonces[idx] }

// HopBeginNonce generates the next unique nonce for BEGIN commands at hop idx
func (c *Circuit) HopBeginNonce(idx int) value_object.Nonce {
	var nonce value_object.Nonce
	nonce = c.baseNonces[idx]
	
	// XOR begin counter into last 8 bytes
	counter := c.beginCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	c.beginCounter[idx]++
	return nonce
}

// HopBeginNoncePeek returns the next nonce without incrementing counter
func (c *Circuit) HopBeginNoncePeek(idx int) value_object.Nonce {
	var nonce value_object.Nonce
	nonce = c.baseNonces[idx]
	
	// XOR begin counter into last 8 bytes
	counter := c.beginCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	return nonce
}

// HopDataNonce generates the next unique nonce for DATA commands at hop idx
func (c *Circuit) HopDataNonce(idx int) value_object.Nonce {
	var nonce value_object.Nonce
	nonce = c.baseNonces[idx]
	
	// XOR data counter into last 8 bytes
	counter := c.dataCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	c.dataCounter[idx]++
	return nonce
}

// HopDataNoncePeek returns the next nonce without incrementing counter
func (c *Circuit) HopDataNoncePeek(idx int) value_object.Nonce {
	var nonce value_object.Nonce
	nonce = c.baseNonces[idx]
	
	// XOR data counter into last 8 bytes
	counter := c.dataCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	return nonce
}

// HopUpstreamDataNonce generates the next unique nonce for upstream DATA commands at hop idx
func (c *Circuit) HopUpstreamDataNonce(idx int) value_object.Nonce {
	var nonce value_object.Nonce
	nonce = c.baseNonces[idx]
	
	// XOR upstream data counter into last 8 bytes
	counter := c.upstreamDataCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	c.upstreamDataCounter[idx]++
	return nonce
}

// HopUpstreamDataNoncePeek returns the next upstream nonce without incrementing counter
func (c *Circuit) HopUpstreamDataNoncePeek(idx int) value_object.Nonce {
	var nonce value_object.Nonce
	nonce = c.baseNonces[idx]
	
	// XOR upstream data counter into last 8 bytes
	counter := c.upstreamDataCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}
	
	return nonce
}

func (c *Circuit) RSAPrivate() *rsa.PrivateKey         { return c.priv }
func (c *Circuit) RSAPublic() *rsa.PublicKey {
	if c.priv == nil {
		return nil
	}
	return &c.priv.PublicKey
}

// WipeKeys zeroes all symmetric keys and forgets the RSA private key.
func (c *Circuit) WipeKeys() {
	for i := range c.keys {
		c.keys[i] = value_object.AESKey{}
	}
	for i := range c.baseNonces {
		c.baseNonces[i] = value_object.Nonce{}
	}
	c.priv = nil
}

// ----------------------------------------------------------------------------
// ストリーム管理

func (c *Circuit) OpenStream() (*StreamState, error) {
	c.strmMu.Lock()
	defer c.strmMu.Unlock()

	sid := value_object.NewStreamIDAuto()
	state := &StreamState{ID: sid}
	c.stream[sid] = state
	return state, nil
}

func (c *Circuit) CloseStream(id value_object.StreamID) {
	c.strmMu.Lock()
	defer c.strmMu.Unlock()
	if st, ok := c.stream[id]; ok {
		st.Closed = true
	}
}

func (c *Circuit) ActiveStreams() []value_object.StreamID {
	c.strmMu.RLock()
	defer c.strmMu.RUnlock()
	out := make([]value_object.StreamID, 0, len(c.stream))
	for id, st := range c.stream {
		if !st.Closed {
			out = append(out, id)
		}
	}
	return out
}

// Conn returns the connection for the given hop index.
func (c *Circuit) Conn(i int) net.Conn {
	if i < len(c.conns) {
		return c.conns[i]
	}
	return nil
}

// SetConn stores the connection for a hop.
func (c *Circuit) SetConn(i int, cconn net.Conn) {
	if i < len(c.conns) {
		c.conns[i] = cconn
	}
}

// ----------------------------------------------------------------------------
// デバッグ表現

func (c *Circuit) String() string {
	return fmt.Sprintf("Circuit(%s) hops=%d streams=%d",
		c.id, len(c.hops), len(c.stream))
}
