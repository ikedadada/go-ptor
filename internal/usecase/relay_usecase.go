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
	Handle(up net.Conn, cell entity.Cell) error
}

type relayUsecaseImpl struct {
	priv   *rsa.PrivateKey
	repo   repoif.CircuitTableRepository
	crypto service.CryptoService
}

func NewRelayUseCase(priv *rsa.PrivateKey, repo repoif.CircuitTableRepository, c service.CryptoService) RelayUseCase {
	return &relayUsecaseImpl{priv: priv, repo: repo, crypto: c}
}

func (uc *relayUsecaseImpl) Handle(up net.Conn, cell entity.Cell) error {
	st, err := uc.repo.Find(cell.CircID)
	switch {
	case errors.Is(err, repoif.ErrNotFound) && cell.End:
		// End for an unknown circuit is ignored
		return nil
	case errors.Is(err, repoif.ErrNotFound) && cell.StreamID.UInt16() == 0:
		// new circuit request
		return uc.extend(up, cell)
	case err != nil:
		return err
	}

	if cell.End {
		// graceful shutdown of an existing circuit
		_ = uc.repo.Delete(cell.CircID)
		return nil
	}

	if len(cell.Data) < 12 {
		// ignore malformed payload
		return nil
	}

	var nonce [12]byte
	copy(nonce[:], cell.Data[:12])
	dec, err := uc.crypto.AESOpen(st.Key(), nonce, cell.Data[12:])
	if err != nil {
		return err
	}
	_, err = st.Down().Write(dec)
	return err
}

func (uc *relayUsecaseImpl) extend(up net.Conn, cell entity.Cell) error {
	p, err := value_object.DecodeExtendPayload(cell.Data)
	if err != nil {
		return err
	}
	dec, err := uc.crypto.RSADecrypt(uc.priv, p.EncKey)
	if err != nil {
		return err
	}
	if len(dec) < 32 {
		return nil
	}
	key, err := value_object.AESKeyFrom(dec[:32])
	if err != nil {
		return err
	}
	down, err := net.Dial("tcp", p.NextHop)
	if err != nil {
		return err
	}
	st := entity.NewConnState(key, up, down)
	if err := uc.repo.Add(cell.CircID, st); err != nil {
		return err
	}
	return sendAck(up, cell.CircID)
}

func sendAck(w net.Conn, cid value_object.CircuitID) error {
	var buf [20]byte
	copy(buf[:16], cid.Bytes())
	binary.BigEndian.PutUint16(buf[18:20], 0)
	_, err := w.Write(buf[:])
	return err
}
