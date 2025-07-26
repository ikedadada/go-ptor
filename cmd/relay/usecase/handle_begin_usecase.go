package usecase

import (
	"fmt"
	"log"
	"net"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// HandleBeginUseCase handles stream start operations
type HandleBeginUseCase interface {
	// Begin starts a new stream
	Begin(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error
}

type handleBeginUseCaseImpl struct {
	repo   repository.ConnStateRepository
	crypto service.CryptoService
	sender service.CellSenderService
}

// NewHandleBeginUseCase creates a new begin use case
func NewHandleBeginUseCase(repo repository.ConnStateRepository, crypto service.CryptoService, sender service.CellSenderService) HandleBeginUseCase {
	return &handleBeginUseCaseImpl{
		repo:   repo,
		crypto: crypto,
		sender: sender,
	}
}

func (uc *handleBeginUseCaseImpl) Begin(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell, ensureServeDown func(*entity.ConnState)) error {
	nonce := st.BeginNonce()
	log.Printf("begin decrypt cid=%s nonce=%x key=%x payloadLen=%d", cid.String(), nonce, st.Key(), len(cell.Payload))
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Payload)
	if err != nil {
		log.Printf("AESOpen begin failed cid=%s nonce=%x error=%v", cid.String(), nonce, err)
		return fmt.Errorf("AESOpen begin cid=%s: %w", cid.String(), err)
	}
	log.Printf("begin decrypt success cid=%s decryptedLen=%d", cid.String(), len(dec))

	if st.IsHidden() {
		p, err := vo.DecodeBeginPayload(dec)
		if err != nil {
			return err
		}
		sid, err := vo.StreamIDFrom(p.StreamID)
		if err != nil {
			return err
		}
		go uc.forwardUpstream(st, cid, sid, st.Down())
		return uc.sender.SendAck(st.Up(), cid)
	}

	if st.Down() != nil {
		ensureServeDown(st)
		c := &entity.Cell{Cmd: vo.CmdBegin, Version: vo.ProtocolV1, Payload: dec}
		return uc.sender.ForwardCell(st.Down(), cid, c)
	}

	p, err := vo.DecodeBeginPayload(dec)
	if err != nil {
		return err
	}
	sid, err := vo.StreamIDFrom(p.StreamID)
	if err != nil {
		return err
	}
	down, err := net.Dial("tcp", p.Target)
	if err != nil {
		c := &entity.Cell{Cmd: vo.CmdDestroy, Version: vo.ProtocolV1}
		_ = uc.sender.ForwardCell(st.Up(), cid, c)
		log.Printf("dial begin target cid=%s addr=%s err=%v", cid.String(), p.Target, err)
		return err
	}
	if err := uc.repo.AddStream(cid, sid, down); err != nil {
		down.Close()
		return err
	}
	ack := &entity.Cell{Cmd: vo.CmdBeginAck, Version: vo.ProtocolV1}
	if err := uc.sender.ForwardCell(st.Up(), cid, ack); err != nil {
		return err
	}
	go uc.forwardUpstream(st, cid, sid, down)
	return nil
}

func (uc *handleBeginUseCaseImpl) forwardUpstream(st *entity.ConnState, cid vo.CircuitID, sid vo.StreamID, down net.Conn) {
	defer down.Close()
	buf := make([]byte, entity.MaxPayloadSize)
	for {
		n, err := down.Read(buf)
		if n > 0 {
			// Use upstream-specific nonce for upstream data encryption
			nonce := st.UpstreamDataNonce()
			log.Printf("upstream encrypt cid=%s nonce=%x", cid.String(), nonce)
			enc, err2 := uc.crypto.AESSeal(st.Key(), nonce, buf[:n])
			if err2 == nil {
				payload, err3 := vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid.UInt16(), Data: enc})
				if err3 == nil {
					c := &entity.Cell{Cmd: vo.CmdData, Version: vo.ProtocolV1, Payload: payload}
					_ = uc.sender.ForwardCell(st.Up(), cid, c)
				}
			}
		}
		if err != nil {
			if sid != 0 {
				_ = uc.repo.RemoveStream(cid, sid)
			}
			endPayload := []byte{}
			if sid != 0 {
				endPayload, _ = vo.EncodeDataPayload(&vo.DataPayload{StreamID: sid.UInt16()})
			}
			_ = uc.sender.ForwardCell(st.Up(), cid, &entity.Cell{Cmd: vo.CmdEnd, Version: vo.ProtocolV1, Payload: endPayload})
			return
		}
	}
}
