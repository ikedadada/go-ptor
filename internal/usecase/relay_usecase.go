package usecase

import (
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"net"

	"ikedadada/go-ptor/internal/domain/entity"
	repoif "ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// RelayUseCase processes cells for a single relay connection.
type RelayUseCase interface {
	Handle(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error
}

type relayUsecaseImpl struct {
	priv   *rsa.PrivateKey
	repo   repoif.CircuitTableRepository
	crypto service.CryptoService
}

// NewRelayUseCase returns a use case to process relay connections.
func NewRelayUseCase(priv *rsa.PrivateKey, repo repoif.CircuitTableRepository, c service.CryptoService) RelayUseCase {
	return &relayUsecaseImpl{priv: priv, repo: repo, crypto: c}
}

func (uc *relayUsecaseImpl) Handle(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
	st, err := uc.repo.Find(cid)
	switch {
	case errors.Is(err, repoif.ErrNotFound) && cell.Cmd == value_object.CmdEnd:
		// End for an unknown circuit is ignored
		return nil
	case errors.Is(err, repoif.ErrNotFound) && cell.Cmd == value_object.CmdExtend:
		// new circuit request
		return uc.extend(up, cid, cell)
	case err != nil:
		return err
	}

	switch cell.Cmd {
	case value_object.CmdEnd:
		_ = uc.repo.Delete(cid)
		return nil
	case value_object.CmdConnect:
		return uc.connect(st, cid, cell)
	case value_object.CmdData:
		dec, err := uc.crypto.AESOpen(st.Key(), st.Nonce(), cell.Payload)
		if err != nil {
			return err
		}
		_, err = st.Down().Write(dec)
		return err
	default:
		return nil
	}
}

func (uc *relayUsecaseImpl) connect(st *entity.ConnState, cid value_object.CircuitID, cell *value_object.Cell) error {
	addr := "127.0.0.1:5003"
	if len(cell.Payload) > 0 {
		p, err := value_object.DecodeConnectPayload(cell.Payload)
		if err != nil {
			return err
		}
		if p.Target != "" {
			addr = p.Target
		}
	}

	down, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	if st.Down() != nil {
		st.Down().Close()
	}
	newSt := entity.NewConnState(st.Key(), st.Nonce(), st.Up(), down)
	if err := uc.repo.Add(cid, newSt); err != nil {
		down.Close()
		return err
	}
	return sendAck(newSt.Up(), cid)
}

func (uc *relayUsecaseImpl) extend(up net.Conn, cid value_object.CircuitID, cell *value_object.Cell) error {
	p, err := value_object.DecodeExtendPayload(cell.Payload)
	if err != nil {
		return err
	}
	dec, err := uc.crypto.RSADecrypt(uc.priv, p.EncKey)
	if err != nil {
		return err
	}
	if len(dec) < 44 {
		return nil
	}
	key, err := value_object.AESKeyFrom(dec[:32])
	if err != nil {
		return err
	}
	nonce, err := value_object.NonceFrom(dec[32:44])
	if err != nil {
		return err
	}
	down, err := net.Dial("tcp", p.NextHop)
	if err != nil {
		return err
	}
	st := entity.NewConnState(key, nonce, up, down)
	if err := uc.repo.Add(cid, st); err != nil {
		return err
	}
	return sendAck(up, cid)
}

func sendAck(w net.Conn, cid value_object.CircuitID) error {
	var buf [20]byte
	copy(buf[:16], cid.Bytes())
	binary.BigEndian.PutUint16(buf[18:20], 0)
	_, err := w.Write(buf[:])
	return err
}
