package usecase

import (
	"fmt"
	"io"
	"log"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// ReceiveAndDecryptDataInput specifies the circuit to receive data from
type ReceiveAndDecryptDataInput struct {
	CircuitID string
}

// DecryptedCellData represents a decrypted cell with its metadata
type DecryptedCellData struct {
	StreamID uint16
	Data     []byte
	Command  vo.CellCommand
}

// ReceiveAndDecryptDataOutput contains the received and decrypted cell data
type ReceiveAndDecryptDataOutput struct {
	CellData    *DecryptedCellData
	IsEOF       bool
	ShouldClose bool
}

// ReceiveAndDecryptDataUseCase handles receiving data from circuit and decrypting onion layers
type ReceiveAndDecryptDataUseCase interface {
	Handle(in ReceiveAndDecryptDataInput) (ReceiveAndDecryptDataOutput, error)
}

type receiveAndDecryptDataUseCaseImpl struct {
	cRepo repository.CircuitRepository
	cSvc  service.CryptoService
	crSvc service.CellReaderService
	peSvc service.PayloadEncodingService
}

// NewReceiveAndDecryptDataUseCase creates a new use case for receiving and decrypting data
func NewReceiveAndDecryptDataUseCase(
	cRepo repository.CircuitRepository,
	cSvc service.CryptoService,
	crSvc service.CellReaderService,
	peSvc service.PayloadEncodingService,
) ReceiveAndDecryptDataUseCase {
	return &receiveAndDecryptDataUseCaseImpl{
		cRepo: cRepo,
		cSvc:  cSvc,
		crSvc: crSvc,
		peSvc: peSvc,
	}
}

func (uc *receiveAndDecryptDataUseCaseImpl) Handle(in ReceiveAndDecryptDataInput) (ReceiveAndDecryptDataOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return ReceiveAndDecryptDataOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}

	cir, err := uc.cRepo.Find(cid)
	if err != nil {
		return ReceiveAndDecryptDataOutput{}, fmt.Errorf("find circuit: %w", err)
	}

	conn := cir.Conn(0)
	if conn == nil {
		return ReceiveAndDecryptDataOutput{}, fmt.Errorf("no connection for circuit")
	}

	// Read next cell from connection
	_, cell, err := uc.crSvc.ReadCell(conn)
	if err != nil {
		if err == io.EOF {
			return ReceiveAndDecryptDataOutput{IsEOF: true}, nil
		}
		return ReceiveAndDecryptDataOutput{}, fmt.Errorf("read cell: %w", err)
	}

	// Handle different cell types
	switch cell.Cmd {
	case vo.CmdData:
		cellData, err := uc.handleDataCell(cell, cir)
		if err != nil {
			log.Printf("handle data cell error: %v", err)
			return ReceiveAndDecryptDataOutput{}, err
		}
		return ReceiveAndDecryptDataOutput{CellData: cellData}, nil

	case vo.CmdEnd:
		cellData, err := uc.handleEndCell(cell)
		if err != nil {
			log.Printf("handle end cell error: %v", err)
			return ReceiveAndDecryptDataOutput{}, err
		}
		return ReceiveAndDecryptDataOutput{
			CellData:    cellData,
			ShouldClose: cellData.StreamID == 0, // Close all if stream ID is 0
		}, nil

	case vo.CmdDestroy:
		return ReceiveAndDecryptDataOutput{ShouldClose: true}, nil

	default:
		log.Printf("unhandled cell command: %v", cell.Cmd)
		return ReceiveAndDecryptDataOutput{}, nil
	}
}

// handleDataCell processes incoming data cells and decrypts onion layers
func (uc *receiveAndDecryptDataUseCaseImpl) handleDataCell(cell *entity.Cell, cir *entity.Circuit) (*DecryptedCellData, error) {
	dp, err := uc.peSvc.DecodeDataPayload(cell.Payload)
	if err != nil {
		return nil, fmt.Errorf("decode data payload: %w", err)
	}

	// Decrypt multi-layer onion encryption
	data, err := uc.decryptOnionLayers(dp.Data, cir)
	if err != nil {
		return nil, fmt.Errorf("onion decryption failed: %w", err)
	}

	return &DecryptedCellData{
		StreamID: dp.StreamID,
		Data:     data,
		Command:  cell.Cmd,
	}, nil
}

// handleEndCell processes stream end commands
func (uc *receiveAndDecryptDataUseCaseImpl) handleEndCell(cell *entity.Cell) (*DecryptedCellData, error) {
	sid := uint16(0)
	if len(cell.Payload) > 0 {
		if p, err := uc.peSvc.DecodeDataPayload(cell.Payload); err == nil {
			sid = p.StreamID
		}
	}

	return &DecryptedCellData{
		StreamID: sid,
		Data:     nil,
		Command:  cell.Cmd,
	}, nil
}

// decryptOnionLayers decrypts multi-layer onion encryption for response data
func (uc *receiveAndDecryptDataUseCaseImpl) decryptOnionLayers(data []byte, cir *entity.Circuit) ([]byte, error) {
	hopCount := len(cir.Hops())

	// Decrypt each layer in reverse circuit order (first hop to exit hop)
	for hop := 0; hop < hopCount; hop++ {
		key := cir.HopKey(hop)
		nonce := cir.HopUpstreamDataNonce(hop)

		decrypted, err := uc.cSvc.AESOpen(key, nonce, data)
		if err != nil {
			return nil, fmt.Errorf("response decrypt failed hop=%d: %w", hop, err)
		}
		data = decrypted
	}

	return data, nil
}
