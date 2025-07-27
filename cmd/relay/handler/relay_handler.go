package handler

import (
	"errors"
	"io"
	"log"
	"net"

	"ikedadada/go-ptor/cmd/relay/usecase"
	"ikedadada/go-ptor/shared/domain/entity"
	"ikedadada/go-ptor/shared/domain/repository"
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"ikedadada/go-ptor/shared/service"
)

// RelayHandler handles relay connections and cell processing
type RelayHandler struct {
	csRepo      repository.ConnStateRepository
	crSvc       service.CellReaderService
	csSvc       service.CellSenderService
	extendUC    usecase.HandleExtendUseCase
	beginUC     usecase.HandleBeginUseCase
	dataUC      usecase.HandleDataUseCase
	endStreamUC usecase.HandleEndStreamUseCase
	destroyUC   usecase.HandleDestroyUseCase
	connectUC   usecase.HandleConnectUseCase
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(
	csRepo repository.ConnStateRepository,
	crSvc service.CellReaderService,
	csSvc service.CellSenderService,
	extendUC usecase.HandleExtendUseCase,
	beginUC usecase.HandleBeginUseCase,
	dataUC usecase.HandleDataUseCase,
	endStreamUC usecase.HandleEndStreamUseCase,
	destroyUC usecase.HandleDestroyUseCase,
	connectUC usecase.HandleConnectUseCase,
) *RelayHandler {
	return &RelayHandler{
		csRepo:      csRepo,
		crSvc:       crSvc,
		csSvc:       csSvc,
		extendUC:    extendUC,
		beginUC:     beginUC,
		dataUC:      dataUC,
		endStreamUC: endStreamUC,
		destroyUC:   destroyUC,
		connectUC:   connectUC,
	}
}

// ServeConn handles a relay connection
func (h *RelayHandler) ServeConn(c net.Conn) {
	log.Printf("ServeConn start local=%s remote=%s", c.LocalAddr(), c.RemoteAddr())
	defer func() {
		_ = c.Close()
		log.Printf("ServeConn stop local=%s remote=%s", c.LocalAddr(), c.RemoteAddr())
	}()

	for {
		cid, cell, err := h.crSvc.ReadCell(c)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			return
		}
		log.Printf("cell cid=%s cmd=%d len=%d", cid.String(), cell.Cmd, len(cell.Payload))
		if err := h.HandleCell(c, cid, cell); err != nil {
			log.Println("handle:", err)
		}
	}
}

// HandleCell routes cells to appropriate handlers
func (h *RelayHandler) HandleCell(up net.Conn, cid vo.CircuitID, cell *entity.Cell) error {
	st, err := h.csRepo.Find(cid)
	switch {
	case errors.Is(err, repository.ErrNotFound) && cell.Cmd == vo.CmdEnd:
		// End for an unknown circuit is ignored
		return nil
	case errors.Is(err, repository.ErrNotFound) && cell.Cmd == vo.CmdExtend:
		// new circuit request
		return h.extendUC.Extend(up, cid, cell)
	case err != nil:
		return err
	}

	switch cell.Cmd {
	case vo.CmdBegin:
		return h.beginUC.Begin(st, cid, cell, h.ensureServeDown)
	case vo.CmdBeginAck:
		return h.csSvc.ForwardCell(st.Up(), cid, cell)
	case vo.CmdEnd:
		return h.endStreamUC.EndStream(st, cid, cell, h.ensureServeDown)
	case vo.CmdDestroy:
		return h.destroyUC.Destroy(st, cid)
	case vo.CmdExtend:
		return h.extendUC.ForwardExtend(st, cid, cell)
	case vo.CmdConnect:
		return h.connectUC.Connect(st, cid, cell, h.ensureServeDown)
	case vo.CmdData:
		return h.dataUC.Data(st, cid, cell, h.ensureServeDown)
	default:
		return nil
	}
}

// ensureServeDown ensures the downstream connection is being served
func (h *RelayHandler) ensureServeDown(st *entity.ConnState) {
	if st == nil || st.Down() == nil || st.IsServed() {
		return
	}
	st.MarkServed()
	go h.ServeConn(st.Down())
}
