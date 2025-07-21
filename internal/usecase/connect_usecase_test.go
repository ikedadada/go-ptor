package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/repository"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

type mockRepoConnect struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockRepoConnect) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockRepoConnect) Save(*entity.Circuit) error             { return nil }
func (m *mockRepoConnect) Delete(value_object.CircuitID) error    { return nil }
func (m *mockRepoConnect) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTxConnect struct {
	cid     value_object.CircuitID
	payload []byte
	err     error
}

func (m *mockTxConnect) TransmitData(value_object.CircuitID, value_object.StreamID, []byte) error {
	return nil
}
func (m *mockTxConnect) InitiateStream(value_object.CircuitID, value_object.StreamID, []byte) error {
	return nil
}
func (m *mockTxConnect) EstablishConnection(c value_object.CircuitID, d []byte) error {
	m.cid = c
	m.payload = d
	return m.err
}
func (m *mockTxConnect) TerminateStream(value_object.CircuitID, value_object.StreamID) error { return nil }
func (m *mockTxConnect) DestroyCircuit(value_object.CircuitID) error                    { return nil }

type connectFactory struct{ tx *mockTxConnect }

func (c connectFactory) New(net.Conn) service.CircuitMessagingService { return c.tx }

func makeTestCircuitConnect() (*entity.Circuit, error) {
	id := value_object.NewCircuitID()
	rid, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := value_object.NewAESKey()
	nonce, _ := value_object.NewNonce()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	return entity.NewCircuit(id, []value_object.RelayID{rid}, []value_object.AESKey{key}, []value_object.Nonce{nonce}, priv)
}

func TestConnectUseCase_Handle(t *testing.T) {
	cir, err := makeTestCircuitConnect()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	cid := cir.ID().String()
	payload, _ := value_object.EncodeConnectPayload(&value_object.ConnectPayload{Target: "x"})

	tests := []struct {
		name  string
		repo  repository.CircuitRepository
		tx    *mockTxConnect
		input usecase.ConnectInput
		err   bool
	}{
		{"ok", &mockRepoConnect{circuit: cir}, &mockTxConnect{}, usecase.ConnectInput{CircuitID: cid, Target: "x"}, false},
		{"circuit not found", &mockRepoConnect{circuit: nil, err: errors.New("nf")}, &mockTxConnect{}, usecase.ConnectInput{CircuitID: cid}, true},
		{"bad id", &mockRepoConnect{}, &mockTxConnect{}, usecase.ConnectInput{CircuitID: "bad"}, true},
		{"tx error", &mockRepoConnect{circuit: cir}, &mockTxConnect{err: errors.New("fail")}, usecase.ConnectInput{CircuitID: cid}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fac := connectFactory{tt.tx}
			crypto := service.NewCryptoService()
			uc := usecase.NewConnectUseCase(tt.repo, fac, crypto)

			// Store expected nonces before use case execution
			k := make([][32]byte, len(cir.Hops()))
			n := make([][12]byte, len(cir.Hops()))
			for i := range cir.Hops() {
				k[i] = cir.HopKey(i)
				n[i] = cir.HopBeginNoncePeek(i)
			}

			_, err := uc.Handle(tt.input)
			if tt.err {
				if err == nil {
					t.Errorf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tt.tx.cid.String() != cid {
				t.Errorf("cid not passed")
			}
			out, err := crypto.AESMultiOpen(k, n, tt.tx.payload)
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}
			if tt.input.Target != "" && string(out) != string(payload) {
				t.Errorf("payload mismatch")
			}
			if tt.input.Target == "" && len(out) != 0 {
				t.Errorf("payload should be empty")
			}
		})
	}
}
