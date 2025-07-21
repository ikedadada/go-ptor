package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"testing"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

type mockCircuitRepoShutdown struct {
	circuit   *entity.Circuit
	findErr   error
	deleteErr error
	deleted   vo.CircuitID
}

func (m *mockCircuitRepoShutdown) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.findErr
}
func (m *mockCircuitRepoShutdown) Save(*entity.Circuit) error { return nil }
func (m *mockCircuitRepoShutdown) Delete(id vo.CircuitID) error {
	m.deleted = id
	return m.deleteErr
}
func (m *mockCircuitRepoShutdown) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTransmitterShutdown struct {
	endCalls []struct {
		cid vo.CircuitID
		sid vo.StreamID
	}
}

func (m *mockTransmitterShutdown) TerminateStream(c vo.CircuitID, s vo.StreamID) error {
	m.endCalls = append(m.endCalls, struct {
		cid vo.CircuitID
		sid vo.StreamID
	}{c, s})
	return nil
}
func (m *mockTransmitterShutdown) InitiateStream(vo.CircuitID, vo.StreamID, []byte) error {
	return nil
}
func (m *mockTransmitterShutdown) TransmitData(c vo.CircuitID, s vo.StreamID, data []byte) error {
	return nil
}
func (m *mockTransmitterShutdown) DestroyCircuit(vo.CircuitID) error              { return nil }
func (m *mockTransmitterShutdown) EstablishConnection(vo.CircuitID, []byte) error { return nil }

type shutdownFactory struct {
	tx service.CircuitMessagingService
}

func (m shutdownFactory) New(net.Conn) service.CircuitMessagingService { return m.tx }

func makeTestCircuitShutdown() (*entity.Circuit, error) {
	id, err := vo.CircuitIDFrom("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	relayID, err := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		return nil, err
	}
	key, err := vo.NewAESKey()
	if err != nil {
		return nil, err
	}
	nonce, err := vo.NewNonce()
	if err != nil {
		return nil, err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	c, err := entity.NewCircuit(id, []vo.RelayID{relayID}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, err
	}
	if _, err := c.OpenStream(); err != nil {
		return nil, err
	}
	if _, err := c.OpenStream(); err != nil {
		return nil, err
	}
	return c, nil
}

func TestShutdownCircuitInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuitShutdown()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	cid := circuit.ID().String()

	t.Run("ok", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{circuit: circuit}
		tx := &mockTransmitterShutdown{}
		uc := usecase.NewShutdownCircuitUsecase(repo, shutdownFactory{tx})
		out, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: cid})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !out.Success {
			t.Errorf("expected success true")
		}
		if repo.deleted.String() != cid {
			t.Errorf("expected repo.Delete called with %s", cid)
		}
		if len(tx.endCalls) < 3 { // 2 streams + control END
			t.Errorf("expected at least 3 TerminateStream calls, got %d", len(tx.endCalls))
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{findErr: errors.New("not found")}
		uc := usecase.NewShutdownCircuitUsecase(repo, shutdownFactory{&mockTransmitterShutdown{}})
		_, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("bad id", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{}
		uc := usecase.NewShutdownCircuitUsecase(repo, shutdownFactory{&mockTransmitterShutdown{}})
		_, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: "bad-uuid"})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		repo := &mockCircuitRepoShutdown{circuit: circuit, deleteErr: errors.New("fail")}
		uc := usecase.NewShutdownCircuitUsecase(repo, shutdownFactory{&mockTransmitterShutdown{}})
		_, err := uc.Handle(usecase.ShutdownCircuitInput{CircuitID: cid})
		if err == nil {
			t.Errorf("expected error")
		}
	})
}
