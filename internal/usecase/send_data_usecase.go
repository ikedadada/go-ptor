package usecase

import (
	"fmt"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// SendDataInput represents application data to forward on a circuit.
type SendDataInput struct {
	CircuitID string
	StreamID  uint16
	Data      []byte
}

// SendDataOutput reports how many bytes were sent.
type SendDataOutput struct {
	BytesSent int `json:"bytes_sent"`
}

// SendDataUseCase forwards data through a circuit.
type SendDataUseCase interface {
	Handle(in SendDataInput) (SendDataOutput, error)
}

type sendDataUseCaseImpl struct {
	cr     repository.CircuitRepository
	tx     service.CircuitTransmitter
	crypto service.CryptoService
}

// NewSendDataUsecase returns a use case for sending data cells.
func NewSendDataUsecase(cr repository.CircuitRepository, tx service.CircuitTransmitter, c service.CryptoService) SendDataUseCase {
	return &sendDataUseCaseImpl{cr: cr, tx: tx, crypto: c}
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

	// onion encrypt payload
	plain := in.Data
	keys := make([][32]byte, 0, len(cir.Hops()))
	nonces := make([][12]byte, 0, len(cir.Hops()))
	for i := range cir.Hops() {
		keys = append(keys, cir.HopKey(i))
		nonces = append(nonces, cir.HopNonce(i))
	}

	enc, err := uc.crypto.AESMultiSeal(keys, nonces, plain)
	if err != nil {
		return SendDataOutput{}, err
	}

	if err := uc.tx.SendData(cid, sid, enc); err != nil {
		return SendDataOutput{}, err
	}
	return SendDataOutput{BytesSent: len(in.Data)}, nil
}
