package usecase_test

import (
	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
	"testing"

	. "github.com/ovechkin-dm/mockio/v2/mock"
)

func TestDecryptCellDataUseCase_Handle(t *testing.T) {
	circuit, _ := makeTestCircuit()

	tests := []struct {
		name                         string
		DecryptCellDataInput         usecase.DecryptCellDataInput
		cSvcAESMultiOpenRes          []byte
		cSvcAESMultiOpenErr          error
		peSvcDecodeDataPayloadRes    *service.DataPayloadDTO
		peSvcDecodeDataPayloadErr    error
		expectsErr                   bool
		expectsDecriptCellDataOutput usecase.DecryptCellDataOutput
	}{
		{
			name: "Data cell",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Cell:    &entity.Cell{Cmd: vo.CmdData, Payload: []byte("test payload")},
				Circuit: circuit,
			},
			cSvcAESMultiOpenRes:       []byte("decrypted data"),
			cSvcAESMultiOpenErr:       nil,
			peSvcDecodeDataPayloadRes: &service.DataPayloadDTO{StreamID: 1, Data: []byte("decrypted data")},
			peSvcDecodeDataPayloadErr: nil,
			expectsErr:                false,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{
				CellData: &usecase.DecryptedCellData{
					StreamID: 1,
					Data:     []byte("decrypted data"),
					Command:  vo.CmdData,
				},
				ShouldClose: false,
			},
		},
		{
			name: "End cell with stream ID = 2",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Cell:    &entity.Cell{Cmd: vo.CmdEnd, Payload: []byte("test payload")},
				Circuit: circuit,
			},
			peSvcDecodeDataPayloadRes: &service.DataPayloadDTO{StreamID: 2, Data: []byte("decrypted data")},
			peSvcDecodeDataPayloadErr: nil,
			expectsErr:                false,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{
				CellData: &usecase.DecryptedCellData{
					StreamID: 2,
					Data:     nil,
					Command:  vo.CmdEnd,
				},
				ShouldClose: false,
			},
		},
		{
			name: "End cell with stream ID == 0",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Cell:    &entity.Cell{Cmd: vo.CmdEnd, Payload: []byte("test payload")},
				Circuit: circuit,
			},
			peSvcDecodeDataPayloadRes: &service.DataPayloadDTO{StreamID: 0, Data: []byte("decrypted data")},
			peSvcDecodeDataPayloadErr: nil,
			expectsErr:                false,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{
				CellData: &usecase.DecryptedCellData{
					StreamID: 0,
					Data:     nil,
					Command:  vo.CmdEnd,
				},
				ShouldClose: true,
			},
		},
		{
			name: "Destory cell",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Cell:    &entity.Cell{Cmd: vo.CmdDestroy, Payload: []byte("test payload")},
				Circuit: &entity.Circuit{},
			},
			expectsErr: false,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{
				ShouldClose: true,
			},
		},
		{
			name: "Relay cell",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Cell:    &entity.Cell{Cmd: vo.CmdBegin, Payload: []byte("test payload")},
				Circuit: &entity.Circuit{},
			},
			expectsErr:                   false,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{},
		},
		{
			name: "nil cell",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Circuit: &entity.Circuit{},
			},
			expectsErr:                   true,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{},
		},
		{
			name: "nil circuit",
			DecryptCellDataInput: usecase.DecryptCellDataInput{
				Cell: &entity.Cell{Cmd: vo.CmdDestroy, Payload: []byte("test payload")},
			},
			expectsErr:                   true,
			expectsDecriptCellDataOutput: usecase.DecryptCellDataOutput{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewMockController(t)

			cSvc := Mock[service.CryptoService](ctrl)
			peSvc := Mock[service.PayloadEncodingService](ctrl)
			// Setup mock behaviors
			if tt.cSvcAESMultiOpenRes != nil || tt.cSvcAESMultiOpenErr != nil {
				When(cSvc.AESMultiOpen(Any[[][32]byte](), Any[[][12]byte](), Any[[]byte]())).ThenReturn(tt.cSvcAESMultiOpenRes, tt.cSvcAESMultiOpenErr)
			}
			if tt.peSvcDecodeDataPayloadRes != nil || tt.peSvcDecodeDataPayloadErr != nil {
				When(peSvc.DecodeDataPayload(Any[[]byte]())).ThenReturn(tt.peSvcDecodeDataPayloadRes, tt.peSvcDecodeDataPayloadErr)
			}

			uc := usecase.NewDecryptCellDataUseCase(cSvc, peSvc) // Use nil services for simplicity

			result, err := uc.Handle(tt.DecryptCellDataInput)

			if tt.expectsErr == (err == nil) {
				t.Errorf("expected error: %v, got: %v", tt.expectsErr, err)
			}

			// useMatchers to check the output
			if result.CellData != nil && tt.expectsDecriptCellDataOutput.CellData != nil {
				if result.CellData.StreamID != tt.expectsDecriptCellDataOutput.CellData.StreamID {
					t.Errorf("expected StreamID %d, got %d", tt.expectsDecriptCellDataOutput.CellData.StreamID, result.CellData.StreamID)
				}
				if string(result.CellData.Data) != string(tt.expectsDecriptCellDataOutput.CellData.Data) {
					t.Errorf("expected Data %s, got %s", tt.expectsDecriptCellDataOutput.CellData.Data, result.CellData.Data)
				}
				if result.CellData.Command != tt.expectsDecriptCellDataOutput.CellData.Command {
					t.Errorf("expected Command %s, got %s", tt.expectsDecriptCellDataOutput.CellData.Command, result.CellData.Command)
				}
			}

			if result.ShouldClose != tt.expectsDecriptCellDataOutput.ShouldClose {
				t.Errorf("expected ShouldClose %v, got %v", tt.expectsDecriptCellDataOutput.ShouldClose, result.ShouldClose)
			}
		})
	}
}
