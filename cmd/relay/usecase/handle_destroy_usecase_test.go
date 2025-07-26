package usecase_test

import (
	"io"
	"net"
	"testing"
	"time"

	repoimpl "ikedadada/go-ptor/cmd/relay/infrastructure/repository"
	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

func TestHandleDestroyUseCase_Destroy(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(time.Second)
	cellSender := service.NewCellSenderService()
	uc := usecase.NewHandleDestroyUseCase(repo, cellSender)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, nil)
	repo.Add(cid, st)

	if err := uc.Destroy(st, cid); err != nil {
		t.Fatalf("destroy error: %v", err)
	}

	// Circuit should be deleted
	if _, err := repo.Find(cid); err == nil {
		t.Errorf("circuit not deleted")
	}

	st.Up().Close()
}

func TestHandleDestroyUseCase_DestroyWithDownstream(t *testing.T) {
	repo := repoimpl.NewConnStateRepository(time.Second)
	cellSender := service.NewCellSenderService()
	uc := usecase.NewHandleDestroyUseCase(repo, cellSender)

	key, _ := vo.NewAESKey()
	nonce, _ := vo.NewNonce()
	cid := vo.NewCircuitID()
	up1, _ := net.Pipe()
	down1, down2 := net.Pipe()

	st := entity.NewConnState(key, nonce, up1, down1)
	repo.Add(cid, st)

	errCh := make(chan error, 1)
	go func() { errCh <- uc.Destroy(st, cid) }()

	// Should forward destroy cell downstream
	buf := make([]byte, 528)
	if _, err := io.ReadFull(down2, buf); err != nil {
		t.Fatalf("read forward: %v", err)
	}
	fwd, err := entity.Decode(buf[16:])
	if err != nil {
		t.Fatalf("decode forward: %v", err)
	}
	if fwd.Cmd != vo.CmdDestroy {
		t.Fatalf("cmd %d", fwd.Cmd)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("destroy error: %v", err)
	}

	// Circuit should be deleted
	if _, err := repo.Find(cid); err == nil {
		t.Errorf("circuit not deleted")
	}

	st.Up().Close()
	st.Down().Close()
}
