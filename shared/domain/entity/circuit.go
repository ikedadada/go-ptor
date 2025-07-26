package entity

import (
	"errors"
	"fmt"
	"net"
	"sync"

	vo "ikedadada/go-ptor/shared/domain/value_object"
)

// MessageType represents the type of message for nonce management
type MessageType int

const (
	MessageTypeBegin MessageType = iota
	MessageTypeData
	MessageTypeUpstreamData
	MessageTypeConnect
)

// ---- StreamState ----------------------------------------------------------

type StreamState struct {
	ID     vo.StreamID
	Closed bool
	// 追加情報が欲しければここに (bytesSent/recv など)
}

// ---- Circuit --------------------------------------------------------------

type Circuit struct {
	id   vo.CircuitID
	hops []vo.RelayID

	keys                map[int]vo.AESKey // per-hop AES key
	baseNonces          map[int]vo.Nonce  // per-hop base Nonce
	beginCounter        map[int]uint64    // per-hop BEGIN counter
	dataCounter         map[int]uint64    // per-hop DATA counter (downstream)
	upstreamDataCounter map[int]uint64    // per-hop upstream DATA counter
	priv                vo.PrivateKey
	conns               []net.Conn
	strmMu              sync.RWMutex
	stream              map[vo.StreamID]*StreamState
}

// NewCircuit は 3 ホップ分の RelayID と鍵束を受け取って生成。
func NewCircuit(id vo.CircuitID, relays []vo.RelayID,
	keys []vo.AESKey, nonces []vo.Nonce, priv vo.PrivateKey) (*Circuit, error) {

	if len(relays) == 0 || len(relays) != len(keys) || len(keys) != len(nonces) {
		return nil, errors.New("hops / keys / nonces length mismatch")
	}
	if priv == nil {
		return nil, errors.New("rsa key required")
	}
	keyMap := make(map[int]vo.AESKey, len(keys))
	ncMap := make(map[int]vo.Nonce, len(nonces))
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
		id:                  id,
		hops:                relays,
		keys:                keyMap,
		baseNonces:          ncMap,
		beginCounter:        beginCounterMap,
		dataCounter:         dataCounterMap,
		upstreamDataCounter: upstreamDataCounterMap,
		priv:                priv,
		conns:               make([]net.Conn, len(relays)),
		stream:              make(map[vo.StreamID]*StreamState),
	}, nil
}

// ----------------------------------------------------------------------------
// 不変部

func (c *Circuit) ID() vo.CircuitID { return c.id }
func (c *Circuit) Hops() []vo.RelayID {
	return append([]vo.RelayID(nil), c.hops...)
}
func (c *Circuit) HopKey(idx int) vo.AESKey      { return c.keys[idx] }
func (c *Circuit) HopBaseNonce(idx int) vo.Nonce { return c.baseNonces[idx] }

// HopBeginNonce generates the next unique nonce for BEGIN commands at hop idx
func (c *Circuit) HopBeginNonce(idx int) vo.Nonce {
	nonce := c.baseNonces[idx]

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
func (c *Circuit) HopBeginNoncePeek(idx int) vo.Nonce {
	nonce := c.baseNonces[idx]

	// XOR begin counter into last 8 bytes
	counter := c.beginCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}

	return nonce
}

// HopDataNonce generates the next unique nonce for DATA commands at hop idx
func (c *Circuit) HopDataNonce(idx int) vo.Nonce {
	nonce := c.baseNonces[idx]

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
func (c *Circuit) HopDataNoncePeek(idx int) vo.Nonce {
	nonce := c.baseNonces[idx]

	// XOR data counter into last 8 bytes
	counter := c.dataCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}

	return nonce
}

// HopUpstreamDataNonce generates the next unique nonce for upstream DATA commands at hop idx
func (c *Circuit) HopUpstreamDataNonce(idx int) vo.Nonce {
	nonce := c.baseNonces[idx]

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
func (c *Circuit) HopUpstreamDataNoncePeek(idx int) vo.Nonce {
	nonce := c.baseNonces[idx]

	// XOR upstream data counter into last 8 bytes
	counter := c.upstreamDataCounter[idx]
	for i := 0; i < 8; i++ {
		nonce[11-i] ^= byte(counter)
		counter >>= 8
	}

	return nonce
}

func (c *Circuit) RSAPrivate() vo.PrivateKey { return c.priv }
func (c *Circuit) RSAPublic() vo.PublicKey {
	if c.priv == nil {
		return nil
	}
	return c.priv.PublicKey()
}

// WipeKeys zeroes all symmetric keys and forgets the RSA private key.
func (c *Circuit) WipeKeys() {
	for i := range c.keys {
		c.keys[i] = vo.AESKey{}
	}
	for i := range c.baseNonces {
		c.baseNonces[i] = vo.Nonce{}
	}
	c.priv = nil
}

// ----------------------------------------------------------------------------
// ストリーム管理

func (c *Circuit) OpenStream() (*StreamState, error) {
	c.strmMu.Lock()
	defer c.strmMu.Unlock()

	sid := vo.NewStreamIDAuto()
	state := &StreamState{ID: sid}
	c.stream[sid] = state
	return state, nil
}

func (c *Circuit) CloseStream(id vo.StreamID) {
	c.strmMu.Lock()
	defer c.strmMu.Unlock()
	if st, ok := c.stream[id]; ok {
		st.Closed = true
	}
}

func (c *Circuit) ActiveStreams() []vo.StreamID {
	c.strmMu.RLock()
	defer c.strmMu.RUnlock()
	out := make([]vo.StreamID, 0, len(c.stream))
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

// GetMessageTypeNonce returns the next nonce for the given message type and hop
func (c *Circuit) GetMessageTypeNonce(hopIndex int, messageType MessageType) vo.Nonce {
	switch messageType {
	case MessageTypeBegin, MessageTypeConnect:
		return c.HopBeginNonce(hopIndex)
	case MessageTypeData:
		return c.HopDataNonce(hopIndex)
	case MessageTypeUpstreamData:
		return c.HopUpstreamDataNonce(hopIndex)
	default:
		return c.HopDataNonce(hopIndex)
	}
}

// IncrementCounter increments the counter for the given message type and hop
func (c *Circuit) IncrementCounter(hopIndex int, messageType MessageType) {
	// Note: The individual nonce methods already increment counters
	// This method is for interface compliance and future extensibility
	switch messageType {
	case MessageTypeBegin, MessageTypeConnect:
		c.beginCounter[hopIndex]++
	case MessageTypeData:
		c.dataCounter[hopIndex]++
	case MessageTypeUpstreamData:
		c.upstreamDataCounter[hopIndex]++
	}
}
