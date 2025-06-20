package value_object

import (
	"fmt"
	"sync/atomic"
)

type StreamID uint16

var streamCounter atomic.Uint32 // 0 は予約値

// NewStreamIDAuto はスレッドセーフに一意な StreamID を生成します。
// 周回後は 1 に戻りますが、uint16 上限 (65535) まで自動インクリメント。
func NewStreamIDAuto() StreamID {
	v := streamCounter.Add(1)
	if v > 0xFFFF {
		streamCounter.Store(1)
		v = 1
	}
	return StreamID(v)
}

// StreamIDFrom は外部値から作成（0 は無効）。
func StreamIDFrom(v uint16) (StreamID, error) {
	if v == 0 {
		return 0, fmt.Errorf("streamID 0 is reserved")
	}
	return StreamID(v), nil
}

func (s StreamID) UInt16() uint16        { return uint16(s) }
func (s StreamID) Equal(o StreamID) bool { return s == o }
