package usecase

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"

	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// HandleExtendUseCase handles circuit extension operations
type HandleExtendUseCase interface {
	// Extend creates a new circuit hop
	Extend(up net.Conn, cid vo.CircuitID, cell *entity.Cell) error
	// ForwardExtend forwards extension to the next hop
	ForwardExtend(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error
}

type handleExtendUseCaseImpl struct {
	priv   vo.PrivateKey
	repo   repository.ConnStateRepository
	crypto service.CryptoService
	sender service.CellSenderService
}

// NewHandleExtendUseCase creates a new extend use case
func NewHandleExtendUseCase(priv vo.PrivateKey, repo repository.ConnStateRepository, crypto service.CryptoService, sender service.CellSenderService) HandleExtendUseCase {
	return &handleExtendUseCaseImpl{
		priv:   priv,
		repo:   repo,
		crypto: crypto,
		sender: sender,
	}
}

func (uc *handleExtendUseCaseImpl) Extend(up net.Conn, cid vo.CircuitID, cell *entity.Cell) error {
	p, err := vo.DecodeExtendPayload(cell.Payload)
	if err != nil {
		log.Printf("decode extend payload cid=%s err=%v", cid.String(), err)
		return err
	}
	relayPriv, relayPub, err := uc.crypto.X25519Generate()
	if err != nil {
		return err
	}
	secret, err := uc.crypto.X25519Shared(relayPriv, p.ClientPub[:])
	if err != nil {
		return err
	}
	key, nonce, err := uc.crypto.DeriveKeyNonce(secret)
	if err != nil {
		return err
	}
	var down net.Conn
	if p.NextHop != "" {
		down, err = net.Dial("tcp", p.NextHop)
		if err != nil {
			log.Printf("dial next hop cid=%s hop=%s err=%v", cid.String(), p.NextHop, err)
			return err
		}
	}
	st := entity.NewConnState(key, nonce, up, down)
	if err := uc.repo.Add(cid, st); err != nil {
		return err
	}
	if down != nil {
		// ServeConn will be started when the next downstream-forwarding command arrives
	}
	createdPayload, err := vo.EncodeCreatedPayload(&vo.CreatedPayload{RelayPub: to32(relayPub)})
	if err != nil {
		return err
	}
	return uc.sender.SendCreated(up, cid, createdPayload)
}

func (uc *handleExtendUseCaseImpl) ForwardExtend(st *entity.ConnState, cid vo.CircuitID, cell *entity.Cell) error {
	if st.Down() == nil {
		return errors.New("no downstream connection")
	}
	if err := uc.sender.ForwardCell(st.Down(), cid, cell); err != nil {
		log.Printf("forward extend cid=%s err=%v", cid.String(), err)
		return err
	}
	var hdr [20]byte
	if _, err := io.ReadFull(st.Down(), hdr[:]); err != nil {
		return err
	}
	l := binary.BigEndian.Uint16(hdr[18:20])
	if l == 0 {
		return errors.New("malformed created payload")
	}
	payload := make([]byte, l)
	if _, err := io.ReadFull(st.Down(), payload); err != nil {
		return err
	}
	return uc.sender.SendCreated(st.Up(), cid, payload)
}

func to32(b []byte) [32]byte {
	var a [32]byte
	copy(a[:], b)
	return a
}
