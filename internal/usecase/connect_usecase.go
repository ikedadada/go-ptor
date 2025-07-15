package usecase

import (
	"fmt"

	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

// ConnectInput triggers a CONNECT cell to the exit relay.
// Target may be empty to use the relay's default hidden service address.
type ConnectInput struct {
	CircuitID string
	Target    string
}

// ConnectOutput reports whether the command was sent.
type ConnectOutput struct {
	Sent bool `json:"sent"`
}

// ConnectUseCase sends a CONNECT cell for the given circuit.
type ConnectUseCase interface {
	Handle(in ConnectInput) (ConnectOutput, error)
}

type connectUsecaseImpl struct {
	repo    repository.CircuitRepository
	factory infraSvc.TransmitterFactory
	crypto  useSvc.CryptoService
}

// NewConnectUseCase creates a use case for CONNECT cells.
func NewConnectUseCase(r repository.CircuitRepository, f infraSvc.TransmitterFactory, c useSvc.CryptoService) ConnectUseCase {
	return &connectUsecaseImpl{repo: r, factory: f, crypto: c}
}

func (uc *connectUsecaseImpl) Handle(in ConnectInput) (ConnectOutput, error) {
	cid, err := value_object.CircuitIDFrom(in.CircuitID)
	if err != nil {
		return ConnectOutput{}, fmt.Errorf("parse circuit id: %w", err)
	}
	cir, err := uc.repo.Find(cid)
	if err != nil {
		return ConnectOutput{}, fmt.Errorf("circuit not found: %w", err)
	}
	payload := []byte{}
	if in.Target != "" {
		payload, err = value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: in.Target})
		if err != nil {
			return ConnectOutput{}, err
		}
	}

	keys := make([][32]byte, 0, len(cir.Hops()))
	nonces := make([][12]byte, 0, len(cir.Hops()))
	for i := range cir.Hops() {
		keys = append(keys, cir.HopKey(i))
		nonces = append(nonces, cir.HopBeginNonce(i))  // CONNECT uses BEGIN nonce
	}

	enc, err := uc.crypto.AESMultiSeal(keys, nonces, payload)
	if err != nil {
		return ConnectOutput{}, err
	}

	conn := cir.Conn(0)
	tx := uc.factory.New(conn)
	if err := tx.SendConnect(cid, enc); err != nil {
		return ConnectOutput{}, err
	}
	return ConnectOutput{Sent: true}, nil
}
