package usecase

import (
	"fmt"
	"log"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// HandleDataUseCase handles data transfer operations
type HandleDataUseCase interface {
	// Data processes data transfer cells
	Data(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error
}

type handleDataUseCaseImpl struct {
	csRepo repository.ConnStateRepository
	cSvc   service.CryptoService
	csSvc  service.CellSenderService
	peSvc  service.PayloadEncodingService
}

// NewHandleDataUseCase creates a new data use case
func NewHandleDataUseCase(csRepo repository.ConnStateRepository, cSvc service.CryptoService, csSvc service.CellSenderService, peSvc service.PayloadEncodingService) HandleDataUseCase {
	return &handleDataUseCaseImpl{
		csRepo: csRepo,
		cSvc:   cSvc,
		csSvc:  csSvc,
		peSvc:  peSvc,
	}
}

func (uc *handleDataUseCaseImpl) Data(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error {
	p, err := uc.peSvc.DecodeDataPayload(cell.Payload)
	if err != nil {
		return err
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}

	// Try to decrypt the data for downstream flow
	nonce := st.DataNonce()
	log.Printf("data decrypt cid=%s nonce=%x key=%x dataLen=%d", cid.String(), nonce, st.Key(), len(p.Data))
	dec, err := uc.cSvc.AESOpen(st.Key(), nonce, p.Data)
	if err != nil {
		// If decryption fails and this is a middle relay, it might be upstream data
		// Add our encryption layer and forward upstream
		if st.Down() != nil {
			log.Printf("data decrypt failed, treating as upstream data cid=%s", cid.String())
			// Add encryption layer for upstream flow
			upstreamNonce := st.UpstreamDataNonce()
			log.Printf("upstream encrypt layer cid=%s nonce=%x", cid.String(), upstreamNonce)
			enc, err2 := uc.cSvc.AESSeal(st.Key(), upstreamNonce, p.Data)
			if err2 != nil {
				log.Printf("upstream encryption failed cid=%s error=%v", cid.String(), err2)
				return err2
			}
			// Forward with additional encryption layer
			upstreamPayload, err3 := uc.peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: p.StreamID, Data: enc})
			if err3 != nil {
				return err3
			}
			upstreamCell := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: upstreamPayload}
			return uc.csSvc.ForwardCell(st.Up(), cid, upstreamCell)
		}
		log.Printf("AESOpen data failed cid=%s nonce=%x error=%v", cid.String(), nonce, err)
		return fmt.Errorf("AESOpen data cid=%s: %w", cid.String(), err)
	}
	log.Printf("data decrypt success cid=%s decryptedLen=%d", cid.String(), len(dec))

	if st.IsHidden() {
		_, err := st.Down().Write(dec)
		return err
	}

	// middle relay: forward downstream with one layer removed
	if st.Down() != nil {
		ensureServeDown(st)
		payload, err := uc.peSvc.EncodeDataPayload(&service.DataPayloadDTO{StreamID: p.StreamID, Data: dec})
		if err != nil {
			return err
		}
		c := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}
		return uc.csSvc.ForwardCell(st.Down(), cid, c)
	}

	// exit relay: write plaintext to the local stream
	conn, err := uc.csRepo.GetStream(cid, sid)
	if err != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = uc.csSvc.ForwardCell(st.Up(), cid, c)
		return nil
	}
	if _, err := conn.Write(dec); err != nil {
		_ = uc.csRepo.RemoveStream(cid, sid)
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = uc.csSvc.ForwardCell(st.Up(), cid, c)
		return err
	}
	return nil
}
