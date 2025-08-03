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
	StreamID vo.StreamID
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
	if in.Cell == nil {
		return DecryptCellDataOutput{}, fmt.Errorf("nil cell")
	}
	if in.Circuit == nil {
		return DecryptCellDataOutput{}, fmt.Errorf("nil circuit")
	}
	switch in.Cell.Cmd {
	case vo.CmdData:
		dp, err := uc.peSvc.DecodeDataPayload(in.Cell.Payload)
		if err != nil {
			log.Printf("decode data payload error: %v", err)
			return DecryptCellDataOutput{}, fmt.Errorf("decode data payload: %w", err)
		}

		// Decrypt multi-layer onion encryption
		hopCount := len(in.Circuit.Hops())

		keys := make([][32]byte, hopCount)
		nonces := make([][12]byte, hopCount)

		// Collect keys and nonces for each hop
		for hop := 0; hop < hopCount; hop++ {
			keys[hop] = in.Circuit.HopKey(hop)
			nonces[hop] = in.Circuit.HopUpstreamDataNonce(hop)
		}
		data, err := uc.cSvc.AESMultiOpen(keys, nonces, dp.Data)

		if err != nil {
			log.Printf("decrypt onion layers error: %v", err)
			return DecryptCellDataOutput{}, fmt.Errorf("onion decryption failed: %w", err)
		}

		return DecryptCellDataOutput{CellData: &DecryptedCellData{
			StreamID: dp.StreamID,
			Data:     data,
			Command:  vo.CmdData,
		}}, nil

	case vo.CmdEnd:
		sid := vo.NewStreamIDControl()
		if len(in.Cell.Payload) > 0 {
			if p, err := uc.peSvc.DecodeDataPayload(in.Cell.Payload); err == nil {
				sid = p.StreamID
			} else {
				log.Printf("decode data payload for end cell error: %v", err)
				return DecryptCellDataOutput{}, fmt.Errorf("decode data payload for end cell: %w", err)
			}
		}
		return DecryptCellDataOutput{
			CellData: &DecryptedCellData{
				StreamID: sid,
				Data:     nil,
				Command:  vo.CmdEnd,
			},
			ShouldClose: sid.Equal(0), // Close all if stream ID is 0
		}, nil

	case vo.CmdDestroy:
		return DecryptCellDataOutput{ShouldClose: true}, nil

	default:
		log.Printf("unhandled cell command: %v", in.Cell.Cmd)
		return DecryptCellDataOutput{}, nil
	}
}
