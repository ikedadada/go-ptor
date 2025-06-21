package service

import "ikedadada/go-ptor/internal/domain/value_object"

// CircuitTransmitter はセル転送を担当するインフラ側ポート。
// 回路 ID + ストリーム ID + データを受け取り、セル化してネットワークに送る。
type CircuitTransmitter interface {
	SendData(c value_object.CircuitID, s value_object.StreamID, data []byte) error
	SendEnd(c value_object.CircuitID, s value_object.StreamID) error
	SendDestroy(c value_object.CircuitID) error
}
