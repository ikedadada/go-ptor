package usecase

import (
	"fmt"
	"log"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// SendDataInput represents application data to forward on a circuit.
type SendDataInput struct {
	CircuitID string
	StreamID  uint16
	Data      []byte
	Cmd       vo.CellCommand // default CmdData
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
	cRepo repository.CircuitRepository
	cSvc  service.CryptoService
	peSvc service.PayloadEncodingService
}

// NewSendDataUseCase returns a use case for sending data cells.
func NewSendDataUseCase(cRepo repository.CircuitRepository, cSvc service.CryptoService, peSvc service.PayloadEncodingService) SendDataUseCase {
	return &sendDataUseCaseImpl{cRepo: cRepo, cSvc: cSvc, peSvc: peSvc}
}

func (uc *sendDataUseCaseImpl) Handle(in SendDataInput) (SendDataOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return SendDataOutput{}, err
	}
	sid, err := vo.StreamIDFrom(in.StreamID)
	if err != nil {
		return SendDataOutput{}, err
	}

	// 回路存在確認（データリンクには不要だがバリデーションで利用）
	cir, err := uc.cRepo.Find(cid)
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
		cmd = vo.CmdData
	}
	keys := make([][32]byte, 0, len(cir.Hops()))
	nonces := make([][12]byte, 0, len(cir.Hops()))

	// Generate nonces in normal order for array indexing
	for i := range cir.Hops() {
		keys = append(keys, cir.HopKey(i))
		var nonce vo.Nonce
		if cmd == vo.CmdBegin {
			nonce = cir.HopBeginNonce(i)
		} else {
			nonce = cir.HopDataNonce(i)
		}
		nonces = append(nonces, nonce)
		log.Printf("send encrypt hop=%d cmd=%d nonce=%x key=%x", i, cmd, nonce, cir.HopKey(i))
	}

	log.Printf("multi-seal input cid=%s plainLen=%d", in.CircuitID, len(plain))
	enc, err := uc.cSvc.AESMultiSeal(keys, nonces, plain)
	if err != nil {
		log.Printf("multi-seal failed cid=%s error=%v", in.CircuitID, err)
		return SendDataOutput{}, err
	}
	log.Printf("multi-seal success cid=%s encLen=%d", in.CircuitID, len(enc))

	var payload []byte
	switch cmd {
	case vo.CmdData:
		// DATA commands need gob-encoded DataPayloadDTO
		payload, err = uc.peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: sid.UInt16(), Data: enc})
		if err != nil {
			return SendDataOutput{}, err
		}
	case vo.CmdBegin:
		// BEGIN commands use raw encrypted data directly
		payload = enc
	default:
		return SendDataOutput{}, fmt.Errorf("unsupported cmd: %d", cmd)
	}

	cell, err := entity.NewCell(cmd, payload)
	if err != nil {
		return SendDataOutput{}, err
	}
	if err := cell.SendToConnection(cir.Conn(0), cid); err != nil {
		return SendDataOutput{}, err
	}
	return SendDataOutput{BytesSent: len(in.Data)}, nil
}
