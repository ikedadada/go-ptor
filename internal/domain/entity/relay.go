package entity

import (
	"sync/atomic"
	"time"

	vo "ikedadada/go-ptor/internal/domain/value_object"
)

type RelayStatus uint8

const (
	Offline RelayStatus = iota
	Online
)

// Relay は Aggregate Root
type Relay struct {
	id       vo.RelayID
	endpoint vo.Endpoint
	pubKey   vo.RSAPubKey

	status  atomic.Uint32 // RelayStatus
	success atomic.Uint64 // セル転送成功数
	failure atomic.Uint64 // セル転送失敗数
	updated atomic.Int64  // UnixNano
}

// コンストラクタ
func NewRelay(id vo.RelayID, ep vo.Endpoint, pk vo.RSAPubKey) *Relay {
	r := &Relay{
		id:       id,
		endpoint: ep,
		pubKey:   pk,
	}
	r.status.Store(uint32(Offline))
	return r
}

// 不変な値オブジェクト取り出し
func (r *Relay) ID() vo.RelayID        { return r.id }
func (r *Relay) Endpoint() vo.Endpoint { return r.endpoint }
func (r *Relay) PubKey() vo.RSAPubKey  { return r.pubKey }

// 状態系
func (r *Relay) Status() RelayStatus { return RelayStatus(r.status.Load()) }
func (r *Relay) LastUpdated() time.Time {
	return time.Unix(0, r.updated.Load()).UTC()
}

// 状態変更
func (r *Relay) SetOnline() {
	r.status.Store(uint32(Online))
	r.updated.Store(time.Now().UTC().UnixNano())
}
func (r *Relay) SetOffline() {
	r.status.Store(uint32(Offline))
	r.updated.Store(time.Now().UTC().UnixNano())
}

// メトリクス
func (r *Relay) IncSuccess() { r.success.Add(1) }
func (r *Relay) IncFailure() { r.failure.Add(1) }

func (r *Relay) Stats() (succ, fail uint64) {
	return r.success.Load(), r.failure.Load()
}
