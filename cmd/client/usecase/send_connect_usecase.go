package usecase

import (
	"fmt"

	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// SendConnectInput triggers a CONNECT cell to the exit relay.
// Target may be empty to use the relay's default hidden service address.
type SendConnectInput struct {
	CircuitID string
	Target    string
}

// SendConnectOutput reports whether the command was sent.
type SendConnectOutput struct {
	Sent bool `json:"sent"`
}

// SendConnectUseCase sends a CONNECT cell for the given circuit.
type SendConnectUseCase interface {
	Handle(in SendConnectInput) (SendConnectOutput, error)
}

type sendConnectUseCaseImpl struct {
	cRepo   repository.CircuitRepository
	factory service.MessagingServiceFactory
	cSvc    service.CryptoService
	peSvc   service.PayloadEncodingService
}

// NewSendConnectUseCase creates a use case for CONNECT cells.
func NewSendConnectUseCase(cRepo repository.CircuitRepository, f service.MessagingServiceFactory, cSvc service.CryptoService, peSvc service.PayloadEncodingService) SendConnectUseCase {
	return &sendConnectUseCaseImpl{cRepo: cRepo, factory: f, cSvc: cSvc, peSvc: peSvc}
}

func (uc *sendConnectUseCaseImpl) Handle(in SendConnectInput) (SendConnectOutput, error) {
	cid, err := vo.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return SendConnectOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}
	cir, err := uc.cRepo.Find(cid)
	if err != nil {
		return SendConnectOutput{}, fmt.Errorf("circuit not found: %w", err)
	}
	payload := []byte{}
	if in.Target != "" {
		payload, err = uc.peSvc.EncodeConnectPayload(&service.ConnectPayloadDTO{Target: in.Target})
		if err != nil {
			return SendConnectOutput{}, err
		}
	}

	keys := make([][32]byte, 0, len(cir.Hops()))
	nonces := make([][12]byte, 0, len(cir.Hops()))

	// Generate nonces in normal order for array indexing
	for i := range cir.Hops() {
		keys = append(keys, cir.HopKey(i))
		nonces = append(nonces, cir.HopBeginNonce(i)) // CONNECT uses BEGIN nonce
	}

	enc, err := uc.cSvc.AESMultiSeal(keys, nonces, payload)
	if err != nil {
		return SendConnectOutput{}, err
	}

	conn := cir.Conn(0)
	tx := uc.factory.New(conn)
	if err := tx.EstablishConnection(cid, enc); err != nil {
		return SendConnectOutput{}, err
	}
	return SendConnectOutput{Sent: true}, nil
}
