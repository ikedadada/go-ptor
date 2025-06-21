package main

import (
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"

	"github.com/google/uuid"
	"ikedadada/go-ptor/internal/domain/value_object"
)

const hdr = 20

func main() {
	listen := flag.String("listen", ":5000", "listen address")
	flag.Parse()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("relay listening on", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(c)
	}
}

func handleConn(c net.Conn) {
	defer c.Close()
	for {
		cell, err := readCell(c)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			return
		}
		log.Printf("cell cid=%s sid=%d end=%v len=%d", cell.circID.String(), cell.streamID.UInt16(), cell.end, len(cell.data))
	}
}

type simpleCell struct {
	circID   value_object.CircuitID
	streamID value_object.StreamID
	data     []byte
	end      bool
}

func readCell(r io.Reader) (simpleCell, error) {
	var hdrBuf [hdr]byte
	if _, err := io.ReadFull(r, hdrBuf[:]); err != nil {
		return simpleCell{}, err
	}
	var id uuid.UUID
	copy(id[:], hdrBuf[:16])
	cid, err := value_object.CircuitIDFrom(id.String())
	if err != nil {
		return simpleCell{}, err
	}
	sid, err := value_object.StreamIDFrom(binary.BigEndian.Uint16(hdrBuf[16:18]))
	if err != nil {
		return simpleCell{}, err
	}
	l := binary.BigEndian.Uint16(hdrBuf[18:20])
	if l == 0xFFFF {
		return simpleCell{circID: cid, streamID: sid, end: true}, nil
	}
	buf := make([]byte, int(l))
	if _, err := io.ReadFull(r, buf); err != nil {
		return simpleCell{}, err
	}
	return simpleCell{circID: cid, streamID: sid, data: buf}, nil
}
