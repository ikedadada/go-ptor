package usecase

import (
	"fmt"
	"ikedadada/go-ptor/internal/application/service"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
)

type SendDataInput struct {
	CircuitID string
	StreamID  uint16
	Data      []byte
}

type SendDataOutput struct {
	BytesSent int `json:"bytes_sent"`
}

type SendDataUseCase interface {
	Handle(in SendDataInput) (SendDataOutput, error)
}

type sendDataUseCaseImpl struct {
	cr repository.CircuitRepository
	tx service.CircuitTransmitter
}

func NewSendDataInteractor(cr repository.CircuitRepository, tx service.CircuitTransmitter) SendDataUseCase {
	return &sendDataUseCaseImpl{cr: cr, tx: tx}
}

func (uc *sendDataUseCaseImpl) Handle(in SendDataInput) (SendDataOutput, error) {
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return SendDataOutput{}, err
	}
	sid, err := value_object.StreamIDFrom(in.StreamID)
	if err != nil {
		return SendDataOutput{}, err
	}

	// 回路存在確認（データリンクには不要だがバリデーションで利用）
	cir, err := uc.cr.Find(cid)
	if err != nil {
		return SendDataOutput{}, fmt.Errorf("circuit not found: %w", err)
	}
	// ストリームが Active かを確認
	active := false
	for _, s := range cir.ActiveStreams() {
		if s.Equal(sid) {
			active = true
			break
		}
	}
	if !active {
		return SendDataOutput{}, fmt.Errorf("stream not active")
	}

	// 送信
	if err := uc.tx.SendData(cid, sid, in.Data); err != nil {
		return SendDataOutput{}, err
	}
	return SendDataOutput{BytesSent: len(in.Data)}, nil
}
