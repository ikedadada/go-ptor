package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
        "encoding/pem"
        "errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

        "ikedadada/go-ptor/internal/domain/value_object"
	repoimpl "ikedadada/go-ptor/internal/infrastructure/repository"
	"ikedadada/go-ptor/internal/infrastructure/service"
	"ikedadada/go-ptor/internal/usecase"

	"github.com/google/uuid"
)


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
                cid, cell, err := readCell(c)
                if err != nil {
                        if err != io.EOF {
                                log.Println("read cell:", err)
                        }
                        return
                }
                log.Printf("cell cid=%s cmd=%d len=%d", cid.String(), cell.Cmd, len(cell.Payload))
                if err := uc.Handle(c, cid, cell); err != nil {
                        log.Println("handle:", err)
                }
        }
}

// readCell reads the cell header and returns the payload as-is. The payload
// may still be encrypted; decryption is performed by RelayUseCase.
func readCell(r io.Reader) (value_object.CircuitID, *value_object.Cell, error) {
        var idBuf [16]byte
        if _, err := io.ReadFull(r, idBuf[:]); err != nil {
                return value_object.CircuitID{}, nil, err
        }
        var id uuid.UUID
        copy(id[:], idBuf[:])
        cid, err := value_object.CircuitIDFrom(id.String())
        if err != nil {
                return value_object.CircuitID{}, nil, err
        }
        var cellBuf [value_object.MaxCellSize]byte
        if _, err := io.ReadFull(r, cellBuf[:]); err != nil {
                return value_object.CircuitID{}, nil, err
        }
        cell, err := value_object.Decode(cellBuf[:])
        if err != nil {
                return value_object.CircuitID{}, nil, err
        }
        return cid, cell, nil
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
