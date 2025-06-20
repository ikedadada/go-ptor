package value_object

import "fmt"

const (
	MaxCellSize    = 512
	headerOverhead = 4 // CMD(1)+VER(1)+LEN(2)
	// Data 最大 = 508 (= 512-4)
	MaxDataLen = MaxCellSize - headerOverhead
)

// CellPayload は暗号化前のペイロード生データ
type CellPayload []byte

// NewCellPayload はサイズ上限チェックのみ行い、不変スライスを返します。
func NewCellPayload(b []byte) (CellPayload, error) {
	if len(b) > MaxDataLen {
		return nil, fmt.Errorf("payload too large: %d > %d", len(b), MaxDataLen)
	}
	clone := make([]byte, len(b))
	copy(clone, b)
	return CellPayload(clone), nil
}

func (p CellPayload) Bytes() []byte {
	clone := make([]byte, len(p))
	copy(clone, p)
	return clone
}
