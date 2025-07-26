package service

import (
	"encoding/binary"
	"log"
	"net"

	"ikedadada/go-ptor/shared/domain/entity"
	vo "ikedadada/go-ptor/shared/domain/value_object"
)

type cellSenderServiceImpl struct{}

// NewCellSenderService creates a new CellSenderService
func NewCellSenderService() CellSenderService {
	return &cellSenderServiceImpl{}
}

func (s *cellSenderServiceImpl) SendCreated(w net.Conn, cid vo.CircuitID, payload []byte) error {
	var hdr [20]byte
	copy(hdr[:16], cid.Bytes())
	hdr[16] = byte(vo.CmdCreated)
	hdr[17] = byte(vo.ProtocolV1)
	binary.BigEndian.PutUint16(hdr[18:20], uint16(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	if err != nil {
		return err
	}
	log.Printf("response created cid=%s", cid.String())
	return nil
}

func (s *cellSenderServiceImpl) SendAck(w net.Conn, cid vo.CircuitID) error {
	c := &entity.Cell{Cmd: vo.CmdBeginAck, Version: vo.ProtocolV1}
	if err := s.ForwardCell(w, cid, c); err != nil {
		return err
	}
	log.Printf("response ack cid=%s", cid.String())
	return nil
}

func (s *cellSenderServiceImpl) ForwardCell(w net.Conn, cid vo.CircuitID, cell *entity.Cell) error {
	buf, err := entity.Encode(*cell)
	if err != nil {
		log.Printf("forward encode cid=%s err=%v", cid.String(), err)
		return err
	}
	out := append(cid.Bytes(), buf...)
	_, err = w.Write(out)
	if err != nil {
		log.Printf("forward write cid=%s err=%v", cid.String(), err)
		return err
	}
	log.Printf("response forward cid=%s cmd=%d len=%d", cid.String(), cell.Cmd, len(cell.Payload))
	return nil
}
