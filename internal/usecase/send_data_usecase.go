package usecase

import (
	"fmt"
	"log"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// SendDataInput represents application data to forward on a circuit.
type SendDataInput struct {
	CircuitID string
	StreamID  uint16
	Data      []byte
	Cmd       byte // default CmdData
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
	cr      repository.CircuitRepository
	factory infraSvc.TransmitterFactory
	crypto  useSvc.CryptoService
}

// NewSendDataUsecase returns a use case for sending data cells.
func NewSendDataUsecase(cr repository.CircuitRepository, f infraSvc.TransmitterFactory, c useSvc.CryptoService) SendDataUseCase {
	return &sendDataUseCaseImpl{cr: cr, factory: f, crypto: c}
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
	cmd := in.Cmd
	if cmd == 0 {
		cmd = value_object.CmdData
	}
	keys := make([][32]byte, 0, len(cir.Hops()))
	nonces := make([][12]byte, 0, len(cir.Hops()))
	
	// Generate nonces in normal order for array indexing
	for i := range cir.Hops() {
		keys = append(keys, cir.HopKey(i))
		var nonce value_object.Nonce
		if cmd == value_object.CmdBegin {
			nonce = cir.HopBeginNonce(i)
		} else {
			nonce = cir.HopDataNonce(i)
		}
		nonces = append(nonces, nonce)
		log.Printf("send encrypt hop=%d cmd=%d nonce=%x key=%x", i, cmd, nonce, cir.HopKey(i))
	}

	log.Printf("multi-seal input cid=%s plainLen=%d", in.CircuitID, len(plain))
	enc, err := uc.crypto.AESMultiSeal(keys, nonces, plain)
	if err != nil {
		log.Printf("multi-seal failed cid=%s error=%v", in.CircuitID, err)
		return SendDataOutput{}, err
	}
	log.Printf("multi-seal success cid=%s encLen=%d", in.CircuitID, len(enc))
	tx := uc.factory.New(cir.Conn(0))
	switch cmd {
	case value_object.CmdData:
		if err := tx.SendData(cid, sid, enc); err != nil {
			return SendDataOutput{}, err
		}
	case value_object.CmdBegin:
		if err := tx.SendBegin(cid, sid, enc); err != nil {
			return SendDataOutput{}, err
		}
	default:
		return SendDataOutput{}, fmt.Errorf("unsupported cmd")
	}
	return SendDataOutput{BytesSent: len(in.Data)}, nil
}
