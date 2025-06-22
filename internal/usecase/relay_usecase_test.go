package usecase_test

import (
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	infraSvc "ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"
)

func TestRelayUseCase_ExtendAndForward(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)

	// prepare extend cell
	key, _ := value_object.NewAESKey()
	enc, _ := crypto.RSAEncrypt(&priv.PublicKey, key[:])
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() { ln.Accept() }()
	payload, _ := value_object.EncodeExtendPayload(&value_object.ExtendPayload{NextHop: ln.Addr().String(), EncKey: enc})
	cid := value_object.NewCircuitID()
	cell := entity.Cell{CircID: cid, StreamID: 0, Data: payload}

	up1, _ := net.Pipe()
	go uc.Handle(up1, cell)

	// ensure entry created
	time.Sleep(10 * time.Millisecond)
	st, err := repo.Find(cid)
	if err != nil {
		t.Fatalf("entry not created: %v", err)
	}
	st.Down().Close()
	st.Up().Close()
}

func TestRelayUseCase_EndUnknown(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	repo := repoimpl.NewCircuitTableRepository(time.Second)
	crypto := infraSvc.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, repo, crypto)
	cid := value_object.NewCircuitID()
	cell := entity.Cell{CircID: cid, StreamID: 1, End: true}
	if err := uc.Handle(nil, cell); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
