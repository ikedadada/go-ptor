package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/cmd/client/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

type mockCircuitRepoSend struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockCircuitRepoSend) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockCircuitRepoSend) Save(*entity.Circuit) error             { return nil }
func (m *mockCircuitRepoSend) Delete(vo.CircuitID) error              { return nil }
func (m *mockCircuitRepoSend) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockConnForSendData struct {
	lastWritten []byte
	err         error
}

func (m *mockConnForSendData) Write(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	m.lastWritten = make([]byte, len(p))
	copy(m.lastWritten, p)
	return len(p), nil
}

func (m *mockConnForSendData) Read([]byte) (int, error)         { return 0, nil }
func (m *mockConnForSendData) Close() error                     { return nil }
func (m *mockConnForSendData) LocalAddr() net.Addr              { return nil }
func (m *mockConnForSendData) RemoteAddr() net.Addr             { return nil }
func (m *mockConnForSendData) SetDeadline(time.Time) error      { return nil }
func (m *mockConnForSendData) SetReadDeadline(time.Time) error  { return nil }
func (m *mockConnForSendData) SetWriteDeadline(time.Time) error { return nil }

func TestSendDataInteractor_Handle(t *testing.T) {
	circuit, err := makeTestCircuit()
	if err != nil {
		t.Fatalf("setup circuit: %v", err)
	}
	st, err := circuit.OpenStream()
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	// Set up connection for the circuit
	conn := &mockConnForSendData{}
	circuit.SetConn(0, conn)

	tests := []struct {
		name       string
		cRepo      repository.CircuitRepository
		input      usecase.SendDataInput
		expectsErr bool
	}{
		{"ok", &mockCircuitRepoSend{circuit: circuit}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, false},
		{"begin", &mockCircuitRepoSend{circuit: circuit}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("target"), Cmd: vo.CmdBegin}, false},
		{"circuit not found", &mockCircuitRepoSend{circuit: nil, err: errors.New("not found")}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"bad id", &mockCircuitRepoSend{circuit: nil}, usecase.SendDataInput{CircuitID: "bad-uuid", StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
		{"stream not active", &mockCircuitRepoSend{circuit: &entity.Circuit{}}, usecase.SendDataInput{CircuitID: circuit.ID().String(), StreamID: st.ID.UInt16(), Data: []byte("hello")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cSvc := service.NewCryptoService()
			peSvc := service.NewPayloadEncodingService()
			uc := usecase.NewSendDataUseCase(tt.cRepo, cSvc, peSvc)
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

type recordConn struct {
	data []byte
}

func (r *recordConn) Write(p []byte) (n int, err error) {
	if len(p) >= 16 { // Skip circuit ID
		r.data = p[16:]
	}
	return len(p), nil
}

func (r *recordConn) Read([]byte) (int, error)         { return 0, nil }
func (r *recordConn) Close() error                     { return nil }
func (r *recordConn) LocalAddr() net.Addr              { return nil }
func (r *recordConn) RemoteAddr() net.Addr             { return nil }
func (r *recordConn) SetDeadline(time.Time) error      { return nil }
func (r *recordConn) SetReadDeadline(time.Time) error  { return nil }
func (r *recordConn) SetWriteDeadline(time.Time) error { return nil }

func TestSendData_OnionRoundTrip(t *testing.T) {
	hops := 3
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	ids := make([]vo.RelayID, hops)
	keys := make([]vo.AESKey, hops)
	nonces := make([]vo.Nonce, hops)
	for i := 0; i < hops; i++ {
		ids[i] = relayID
		k, _ := vo.NewAESKey()
		n, _ := vo.NewNonce()
		keys[i] = k
		nonces[i] = n
	}
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	cir, err := entity.NewCircuit(vo.NewCircuitID(), ids, keys, nonces, priv)
	if err != nil {
		t.Fatalf("circuit: %v", err)
	}
	st, _ := cir.OpenStream()

	conn := &recordConn{}
	cir.SetConn(0, conn)

	cRepo := &mockCircuitRepoSend{circuit: cir}
	cSvc := service.NewCryptoService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewSendDataUseCase(cRepo, cSvc, peSvc)
	data := []byte("hello")
	if _, err := uc.Handle(usecase.SendDataInput{CircuitID: cir.ID().String(), StreamID: st.ID.UInt16(), Data: data}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	// Decode cell from written data
	cell, err := entity.Decode(conn.data)
	if err != nil {
		t.Fatalf("decode cell: %v", err)
	}

	// First decode the DataPayloadDTO from the cell payload
	dto, err := peSvc.DecodeDataPayload(cell.Payload)
	if err != nil {
		t.Fatalf("decode DataPayloadDTO: %v", err)
	}

	k2 := make([][32]byte, hops)
	n2 := make([][12]byte, hops)
	for i := 0; i < hops; i++ {
		k2[i] = keys[i]
		n2[i] = nonces[i]
	}
	out, err := cSvc.AESMultiOpen(k2, n2, dto.Data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(data) {
		t.Errorf("round-trip mismatch")
	}
}

func TestSendData_BeginRoundTrip(t *testing.T) {
	hops := 2
	relayID, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	ids := make([]vo.RelayID, hops)
	keys := make([]vo.AESKey, hops)
	nonces := make([]vo.Nonce, hops)
	for i := 0; i < hops; i++ {
		ids[i] = relayID
		k, _ := vo.NewAESKey()
		n, _ := vo.NewNonce()
		keys[i] = k
		nonces[i] = n
	}
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	cir, err := entity.NewCircuit(vo.NewCircuitID(), ids, keys, nonces, priv)
	if err != nil {
		t.Fatalf("circuit: %v", err)
	}
	st, _ := cir.OpenStream()

	conn := &recordConn{}
	cir.SetConn(0, conn)

	cRepo := &mockCircuitRepoSend{circuit: cir}
	cSvc := service.NewCryptoService()
	peSvc := service.NewPayloadEncodingService()
	uc := usecase.NewSendDataUseCase(cRepo, cSvc, peSvc)
	payload, _ := peSvc.EncodeBeginPayload(&service.BeginPayloadDTO{StreamID: st.ID.UInt16(), Target: "example.com:80"})
	if _, err := uc.Handle(usecase.SendDataInput{CircuitID: cir.ID().String(), StreamID: st.ID.UInt16(), Data: payload, Cmd: vo.CmdBegin}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	// Decode cell from written data
	cell, err := entity.Decode(conn.data)
	if err != nil {
		t.Fatalf("decode cell: %v", err)
	}

	k2 := make([][32]byte, hops)
	n2 := make([][12]byte, hops)
	for i := 0; i < hops; i++ {
		k2[i] = keys[i]
		n2[i] = nonces[i]
	}
	out, err := cSvc.AESMultiOpen(k2, n2, cell.Payload)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(out) != string(payload) {
		t.Errorf("round-trip mismatch")
	}
}
