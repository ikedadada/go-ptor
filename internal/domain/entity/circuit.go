package entity

import (
	"errors"
	"fmt"
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

	keys   map[int]value_object.AESKey // per-hop AES key
	nonces map[int]value_object.Nonce  // per-hop Nonce
	strmMu sync.RWMutex
	stream map[value_object.StreamID]*StreamState
}

// NewCircuit は 3 ホップ分の RelayID と鍵束を受け取って生成。
func NewCircuit(id value_object.CircuitID, relays []value_object.RelayID,
	keys []value_object.AESKey, nonces []value_object.Nonce) (*Circuit, error) {

	if len(relays) == 0 || len(relays) != len(keys) || len(keys) != len(nonces) {
		return nil, errors.New("hops / keys / nonces length mismatch")
	}
	keyMap := make(map[int]value_object.AESKey, len(keys))
	ncMap := make(map[int]value_object.Nonce, len(nonces))
	for i := range keys {
		keyMap[i] = keys[i]
		ncMap[i] = nonces[i]
	}
	return &Circuit{
		id:     id,
		hops:   relays,
		keys:   keyMap,
		nonces: ncMap,
		stream: make(map[value_object.StreamID]*StreamState),
	}, nil
}

// ----------------------------------------------------------------------------
// 不変部

func (c *Circuit) ID() value_object.CircuitID { return c.id }
func (c *Circuit) Hops() []value_object.RelayID {
	return append([]value_object.RelayID(nil), c.hops...)
}
func (c *Circuit) HopKey(idx int) value_object.AESKey  { return c.keys[idx] }
func (c *Circuit) HopNonce(idx int) value_object.Nonce { return c.nonces[idx] }

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

// ----------------------------------------------------------------------------
// デバッグ表現

func (c *Circuit) String() string {
	return fmt.Sprintf("Circuit(%s) hops=%d streams=%d",
		c.id, len(c.hops), len(c.stream))
}
