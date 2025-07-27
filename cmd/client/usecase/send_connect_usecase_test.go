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

type mockRepoConnect struct {
	circuit *entity.Circuit
	err     error
}

func (m *mockRepoConnect) Find(id vo.CircuitID) (*entity.Circuit, error) {
	return m.circuit, m.err
}
func (m *mockRepoConnect) Save(*entity.Circuit) error             { return nil }
func (m *mockRepoConnect) Delete(vo.CircuitID) error              { return nil }
func (m *mockRepoConnect) ListActive() ([]*entity.Circuit, error) { return nil, nil }

type mockConnToTestCapture struct {
	lastWritten []byte
	err         error
}

func (m *mockConnToTestCapture) Write(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	m.lastWritten = make([]byte, len(p))
	copy(m.lastWritten, p)
	return len(p), nil
}

func (m *mockConnToTestCapture) Read([]byte) (int, error)         { return 0, nil }
func (m *mockConnToTestCapture) Close() error                     { return nil }
func (m *mockConnToTestCapture) LocalAddr() net.Addr              { return nil }
func (m *mockConnToTestCapture) RemoteAddr() net.Addr             { return nil }
func (m *mockConnToTestCapture) SetDeadline(time.Time) error      { return nil }
func (m *mockConnToTestCapture) SetReadDeadline(time.Time) error  { return nil }
func (m *mockConnToTestCapture) SetWriteDeadline(time.Time) error { return nil }

func makeTestCircuitConnect() (*entity.Circuit, *mockConnToTestCapture, error) {
	id := vo.NewCircuitID()
	rid, _ := vo.NewRelayID("550e8400-e29b-41d4-a716-446655440000")
	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	rawKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	priv := vo.NewRSAPrivKey(rawKey)
	conn := &mockConnToTestCapture{}
	cir, err := entity.NewCircuit(id, []vo.RelayID{rid}, []vo.AESKey{key}, []vo.Nonce{nonce}, priv)
	if err != nil {
		return nil, nil, err
	}
	cir.SetConn(0, conn)
	return cir, conn, nil
}

func TestSendConnectUseCase_Handle(t *testing.T) {
	cir, conn, err := makeTestCircuitConnect()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	cid := cir.ID().String()
	peSvc := service.NewPayloadEncodingService()
	payload, _ := peSvc.EncodeConnectPayload(&service.ConnectPayloadDTO{Target: "x"})

	tests := []struct {
		name  string
		cRepo repository.CircuitRepository
		input usecase.SendConnectInput
		err   bool
	}{
		{"ok", &mockRepoConnect{circuit: cir}, usecase.SendConnectInput{CircuitID: cid, Target: "x"}, false},
		{"circuit not found", &mockRepoConnect{circuit: nil, err: errors.New("nf")}, usecase.SendConnectInput{CircuitID: cid}, true},
		{"bad id", &mockRepoConnect{}, usecase.SendConnectInput{CircuitID: "bad"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cSvc := service.NewCryptoService()
			peSvc := service.NewPayloadEncodingService()
			uc := usecase.NewSendConnectUseCase(tt.cRepo, cSvc, peSvc)

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

			// Verify that data was written to connection
			if len(conn.lastWritten) == 0 {
				t.Errorf("no data written to connection")
				return
			}

			// Extract circuit ID and cell data from written packet
			if len(conn.lastWritten) < 16 {
				t.Errorf("packet too short")
				return
			}
			cellData := conn.lastWritten[16:] // Skip circuit ID bytes

			// Decode and verify cell
			cell, err := entity.Decode(cellData)
			if err != nil {
				t.Fatalf("decode cell: %v", err)
			}
			if cell.Cmd != vo.CmdConnect {
				t.Errorf("expected CONNECT command, got %v", cell.Cmd)
			}

			// Decrypt payload and verify
			out, err := cSvc.AESMultiOpen(k, n, cell.Payload)
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
