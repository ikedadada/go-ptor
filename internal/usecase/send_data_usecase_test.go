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
	useSvc "ikedadada/go-ptor/internal/usecase/service"
)

type mockCircuitRepoSend struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoSend) Find(id value_object.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoSend) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoSend) Delete(value_object.CircuitID) error    { return nil }
func (m *mockCircuitRepoSend) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockTransmitterSend struct{ err error }

func (m *mockTransmitterSend) TransmitData(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return m.err
}
func (m *mockTransmitterSend) InitiateStream(c value_object.CircuitID, s value_object.StreamID, data []byte) error {
	return m.err
}
func (m *mockTransmitterSend) TerminateStream(c value_object.CircuitID, s value_object.StreamID) error {
	return nil
}
func (m *mockTransmitterSend) DestroyCircuit(value_object.CircuitID) error         { return nil }
func (m *mockTransmitterSend) EstablishConnection(value_object.CircuitID, []byte) error { return nil }

type sendFactory struct{ tx useSvc.CircuitMessagingService }

func (m sendFactory) New(net.Conn) useSvc.CircuitMessagingService { return m.tx }

func TestSendDataInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	st, err := circuit.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	tests := []struct {
		name       string
		repo       repository.CircuitRepository
		fac        useSvc.MessagingServiceFactory
		input      usecase.SendDataInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoSend{circuit: circuit}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, false},
		{"begin", &mockCircuitRepoSend{circuit: circuit}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("target"), Cmd: value_object.CmdBegin}, false},
		{"circuit not found", &mockCircuitRepoSend{circuit: nil, err: errors.New("not found")}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"bad id", &mockCircuitRepoSend{circuit: nil}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"tx error", &mockCircuitRepoSend{circuit: circuit}, sendFactory{&mockTransmitterSend{err: errors.New("fail")}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"stream not active", &mockCircuitRepoSend{circuit: &entity.Circuit{}}, sendFactory{&mockTransmitterSend{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecase.NewSendDataUsecase(tt.repo, tt.fac, useSvc.NewCryptoService())
			_, err := uc.Handle(tt.input)
			if tt.expectsErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.expectsErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Additional tests from send_data_roundtrip_test.go

type recordTx struct{ data []byte }

func (r *recordTx) TransmitData(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	r.data = d
	return nil
}
func (r *recordTx) InitiateStream(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	r.data = d
	return nil
}
func (r *recordTx) TerminateStream(value_object.CircuitID, value_object.StreamID) error { return nil }
func (r *recordTx) DestroyCircuit(value_object.CircuitID) error                    { return nil }
func (r *recordTx) EstablishConnection(value_object.CircuitID, []byte) error            { return nil }

type recordFactory struct{ tx *recordTx }

func (m recordFactory) New(net.Conn) useSvc.CircuitMessagingService { return m.tx }

func TestSendData_OnionRoundTrip(t *testing.T) {
	hops := 3
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	ids := make([]value_object.RelayID, hops)
	keys := make([]value_object.AESKey, hops)
	nonces := make([]value_object.Nonce, hops)
	for i := 0; i < hops; i++ {
		ids[i] = relayID
		k, _ := value_object.NewAESKey()
		n, _ := value_object.NewNonce()
		keys[i] = k
		nonces[i] = n
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cir, err := entity.NewCircuit(value_object.NewCircuitID(), ids, keys, nonces, priv)
	if err != nil {
		t.Fatalf("circuit: %v", err)
	}
	st, _ := cir.OpenStream()

	repo := &mockCircuitRepoSend{circuit: cir}
	tx := &recordTx{}
	crypto := useSvc.NewCryptoService()
	uc := usecase.NewSendDataUsecase(repo, recordFactory{tx}, crypto)
	data := []byte("hello")
	if _, err := uc.Handle(usecase.SendDataInput{CircuitID: cir.ID().String(), StreamID: st.ID.UInt16(), Data: data}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	k2 := make([][32]byte, hops)
	n2 := make([][12]byte, hops)
	for i := 0; i < hops; i++ {
		k2[i] = keys[i]
		n2[i] = nonces[i]
	}
	out, err := crypto.AESMultiOpen(k2, n2, tx.data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(data) {
		t.Errorf("round-trip mismatch")
	}
}

func TestSendData_BeginRoundTrip(t *testing.T) {
	hops := 2
	relayID, _ := value_object.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	ids := make([]value_object.RelayID, hops)
	keys := make([]value_object.AESKey, hops)
	nonces := make([]value_object.Nonce, hops)
	for i := 0; i < hops; i++ {
		ids[i] = relayID
		k, _ := value_object.NewAESKey()
		n, _ := value_object.NewNonce()
		keys[i] = k
		nonces[i] = n
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cir, err := entity.NewCircuit(value_object.NewCircuitID(), ids, keys, nonces, priv)
	if err != nil {
		t.Fatalf("circuit: %v", err)
	}
	st, _ := cir.OpenStream()

	repo := &mockCircuitRepoSend{circuit: cir}
	tx := &recordTx{}
	crypto := useSvc.NewCryptoService()
	uc := usecase.NewSendDataUsecase(repo, recordFactory{tx}, crypto)
	payload, _ := value_object.EncodeBeginPayload(&value_object.BeginPayload{StreamID: st.ID.UInt16(), Target: "example.com:80"})
	if _, err := uc.Handle(usecase.SendDataInput{CircuitID: cir.ID().String(), StreamID: st.ID.UInt16(), Data: payload, Cmd: value_object.CmdBegin}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	k2 := make([][32]byte, hops)
	n2 := make([][12]byte, hops)
	for i := 0; i < hops; i++ {
		k2[i] = keys[i]
		n2[i] = nonces[i]
	}
	out, err := crypto.AESMultiOpen(k2, n2, tx.data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(payload) {
		t.Errorf("round-trip mismatch")
	}
}
