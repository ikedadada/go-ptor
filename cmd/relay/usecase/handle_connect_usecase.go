package usecase

import (
	"fmt"
	"log"
	"net"
	"os"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// HandleConnectUseCase handles connection establishment operations
type HandleConnectUseCase interface {
	// Connect establishes connection to hidden service
	Connect(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error
}

type handleConnectUseCaseImpl struct {
	csRepo repository.ConnStateRepository
	cSvc   service.CryptoService
	csSvc  service.CellSenderService
	peSvc  service.PayloadEncodingService
}

// NewHandleConnectUseCase creates a new connect use case
func NewHandleConnectUseCase(csRepo repository.ConnStateRepository, cSvc service.CryptoService, csSvc service.CellSenderService, peSvc service.PayloadEncodingService) HandleConnectUseCase {
	return &handleConnectUseCaseImpl{
		csRepo: csRepo,
		cSvc:   cSvc,
		csSvc:  csSvc,
		peSvc:  peSvc,
	}
}

func (uc *handleConnectUseCaseImpl) Connect(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error {
	// middle relay: peel one layer and forward the remaining ciphertext
	if st.Down() != nil {
		ensureServeDown(st)
		nonce := st.BeginNonce()
		log.Printf("connect decrypt cid=%s nonce=%x", cid.String(), nonce)
		dec, err := uc.cSvc.AESOpen(st.Key(), nonce, cell.Payload)
		if err != nil {
			return fmt.Errorf("AESOpen connect cid=%s: %w", cid.String(), err)
		}
		c := &entity.Cell{Cmd: vo.CmdConnect, Version: vo.ProtocolV1, Payload: dec}
		return uc.csSvc.ForwardCell(st.Down(), cid, c)
	}

	// exit relay: decode final payload and connect to the hidden service
	nonce := st.BeginNonce()
	log.Printf("connect exit decrypt cid=%s nonce=%x", cid.String(), nonce)
	dec, err := uc.cSvc.AESOpen(st.Key(), nonce, cell.Payload)
	if err != nil {
		return fmt.Errorf("AESOpen connect cid=%s: %w", cid.String(), err)
	}
	addr := os.Getenv("PTOR_HIDDEN_ADDR")
	if addr == "" {
		addr = os.Getenv("HIDDEN_ADDR")
	}
	if addr == "" {
		addr = "hidden:5000"
	}
	if len(dec) > 0 {
		p, err := uc.peSvc.DecodeConnectPayload(dec)
		if err != nil {
			return err
		}
		if p.Target != "" {
			addr = p.Target
		}
	}

	down, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("dial hidden cid=%s addr=%s err=%v", cid.String(), addr, err)
		return err
	}
	if st.Down() != nil {
		st.Down().Close()
	}
	beginCounter, dataCounter := st.GetCounters()
	newSt := entity.NewConnStateWithCounters(st.Key(), st.Nonce(), st.Up(), down, beginCounter, dataCounter)
	newSt.SetHidden(true)
	if err := uc.csRepo.Add(cid, newSt); err != nil {
		down.Close()
		return err
	}
	if err := uc.csSvc.SendAck(newSt.Up(), cid); err != nil {
		return err
	}
	return nil
}
