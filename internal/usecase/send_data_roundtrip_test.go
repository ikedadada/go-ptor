package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
	"ikedadada/go-ptor/internal/usecase/service"
)

type recordTx struct{ data []byte }

func (r *recordTx) SendData(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	r.data = d
	return nil
}
func (r *recordTx) SendBegin(c value_object.CircuitID, s value_object.StreamID, d []byte) error {
	r.data = d
	return nil
}
func (r *recordTx) SendEnd(value_object.CircuitID, value_object.StreamID) error { return nil }
func (r *recordTx) SendDestroy(value_object.CircuitID) error                    { return nil }

type recordFactory struct{ tx *recordTx }

func (m recordFactory) New(net.Conn) service.CircuitTransmitter { return m.tx }

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
	crypto := infraSvc.NewCryptoService()
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
	crypto := infraSvc.NewCryptoService()
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
