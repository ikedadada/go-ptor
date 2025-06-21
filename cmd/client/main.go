package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	entry := flag.String("entry", "127.0.0.1:5000", "entry relay address")
	hops := flag.Int("hops", 3, "number of hops")
	dirURL := flag.String("dir", "", "directory service URL")
	flag.Parse()

	fmt.Printf("building circuit via %s with %d hops (dir=%s)\n", *entry, *hops, *dirURL)
	cir, err := buildCircuit(*hops)
	if err != nil {
		log.Fatal(err)
	}
	defer cir.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:9050")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("SOCKS5 proxy listening on", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go HandleSOCKS(c, cir.Dial)
	}
}

// ---- minimal circuit stub ----------------------------------------------

type circuit struct{}

func buildCircuit(hops int) (*circuit, error) {
	// In a full implementation this would establish multi-hop encryption.
	fmt.Printf("circuit constructed with %d hops\n", hops)
	return &circuit{}, nil
}

func (c *circuit) Dial(addr string) (net.Conn, error) { return net.Dial("tcp", addr) }
func (c *circuit) Close() error                       { return nil }

// ---- minimal SOCKS5 handler --------------------------------------------

func HandleSOCKS(conn net.Conn, dial func(string) (net.Conn, error)) {
	defer conn.Close()

	var buf [262]byte
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	n := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:n]); err != nil {
		return
	}
	conn.Write([]byte{5, 0})

	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[1] != 1 {
		return
	}
	var host string
	switch buf[3] {
	case 1:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 3:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			return
		}
		host = string(buf[:l])
	default:
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	addr := fmt.Sprintf("%s:%d", host, port)

	target, err := dial(addr)
	if err != nil {
		conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer target.Close()
	conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})

	go io.Copy(target, conn)
	io.Copy(conn, target)
}
