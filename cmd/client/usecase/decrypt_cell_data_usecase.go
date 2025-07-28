package usecase

import (
	"fmt"
	"log"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// DecryptCellDataInput contains the cell and circuit for decryption
type DecryptCellDataInput struct {
	Cell    *entity.Cell
	Circuit *entity.Circuit
}

// DecryptedCellData represents a decrypted cell with its metadata
type DecryptedCellData struct {
	StreamID uint16
	Data     []byte
	Command  vo.CellCommand
}

// DecryptCellDataOutput contains the decrypted cell data
type DecryptCellDataOutput struct {
	CellData    *DecryptedCellData
	ShouldClose bool
}

// DecryptCellDataUseCase handles decrypting cell data and processing different cell types
type DecryptCellDataUseCase interface {
	Handle(in DecryptCellDataInput) (DecryptCellDataOutput, error)
}

type decryptCellDataUseCaseImpl struct {
	cSvc  service.CryptoService
	peSvc service.PayloadEncodingService
}

// NewDecryptCellDataUseCase creates a new use case for decrypting cell data
func NewDecryptCellDataUseCase(
	cSvc service.CryptoService,
	peSvc service.PayloadEncodingService,
) DecryptCellDataUseCase {
	return &decryptCellDataUseCaseImpl{
		cSvc:  cSvc,
		peSvc: peSvc,
	}
}

func (uc *decryptCellDataUseCaseImpl) Handle(in DecryptCellDataInput) (DecryptCellDataOutput, error) {
	// Handle different cell types
	switch in.Cell.Cmd {
	case vo.CmdData:
		cellData, err := uc.handleDataCell(in.Cell, in.Circuit)
		if err != nil {
			log.Printf("handle data cell error: %v", err)
			return DecryptCellDataOutput{}, err
		}
		return DecryptCellDataOutput{CellData: cellData}, nil

	case vo.CmdEnd:
		cellData, err := uc.handleEndCell(in.Cell)
		if err != nil {
			log.Printf("handle end cell error: %v", err)
			return DecryptCellDataOutput{}, err
		}
		return DecryptCellDataOutput{
			CellData:    cellData,
			ShouldClose: cellData.StreamID == 0, // Close all if stream ID is 0
		}, nil

	case vo.CmdDestroy:
		return DecryptCellDataOutput{ShouldClose: true}, nil

	default:
		log.Printf("unhandled cell command: %v", in.Cell.Cmd)
		return DecryptCellDataOutput{}, nil
	}
}

// handleDataCell processes incoming data cells and decrypts onion layers
func (uc *decryptCellDataUseCaseImpl) handleDataCell(cell *entity.Cell, cir *entity.Circuit) (*DecryptedCellData, error) {
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
func (uc *decryptCellDataUseCaseImpl) handleEndCell(cell *entity.Cell) (*DecryptedCellData, error) {
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
func (uc *decryptCellDataUseCaseImpl) decryptOnionLayers(data []byte, cir *entity.Circuit) ([]byte, error) {
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
