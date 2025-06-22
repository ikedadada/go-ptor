package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	"ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"

	"github.com/google/uuid"
)

const hdr = 20

func main() {
	listen := flag.String("listen", ":5000", "listen address")
	privPath := flag.String("priv", "", "RSA private key")
	flag.Parse()
	var priv *rsa.PrivateKey
	var err error
	if *privPath == "" {
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		priv, err = loadRSAPriv(*privPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	tbl := repoimpl.NewCircuitTableRepository(time.Minute)
	cryptoSvc := service.NewCryptoService()
	uc := usecase.NewRelayUseCase(priv, tbl, cryptoSvc)

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
		go handleConn(c, uc)
	}
}

func handleConn(c net.Conn, uc usecase.RelayUseCase) {
	defer c.Close()
	for {
		cell, err := readCell(c)
		if err != nil {
			if err != io.EOF {
				log.Println("read cell:", err)
			}
			return
		}
		log.Printf("cell cid=%s sid=%d end=%v len=%d", cell.CircID.String(), cell.StreamID.UInt16(), cell.End, len(cell.Data))
		if err := uc.Handle(c, cell); err != nil {
			log.Println("handle:", err)
		}
	}
}

// readCell reads the cell header and returns the payload as-is. The payload
// may still be encrypted; decryption is performed by RelayUseCase.
func readCell(r io.Reader) (entity.Cell, error) {
	var hdrBuf [hdr]byte
	if _, err := io.ReadFull(r, hdrBuf[:]); err != nil {
		return entity.Cell{}, err
	}
	var id uuid.UUID
	copy(id[:], hdrBuf[:16])
	cid, err := value_object.CircuitIDFrom(id.String())
	if err != nil {
		return entity.Cell{}, err
	}
	sid, err := value_object.StreamIDFrom(binary.BigEndian.Uint16(hdrBuf[16:18]))
	if err != nil {
		return entity.Cell{}, err
	}
	l := binary.BigEndian.Uint16(hdrBuf[18:20])
	if l == 0xFFFF {
		return entity.Cell{CircID: cid, StreamID: sid, End: true}, nil
	}
	buf := make([]byte, int(l))
	if _, err := io.ReadFull(r, buf); err != nil {
		return entity.Cell{}, err
	}
	return entity.Cell{CircID: cid, StreamID: sid, Data: buf}, nil
}

// loadRSAPriv loads an RSA private key from a PEM file.
func loadRSAPriv(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, io.ErrUnexpectedEOF
	}
	switch blk.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(blk.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not RSA private key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported key type %q", blk.Type)
	}
}
