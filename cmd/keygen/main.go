package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
)

func saveRSAPriv(path string, key *rsa.PrivateKey) error {
	b := x509.MarshalPKCS1PrivateKey(key)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})
	return os.WriteFile(path, pemData, 0600)
}

func saveRSAPub(path string, key *rsa.PublicKey) error {
	b := x509.MarshalPKCS1PublicKey(key)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: b})
	return os.WriteFile(path, pemData, 0644)
}

func main() {
	out := flag.String("out", "rsa_key.pem", "output private key file")
	flag.Parse()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}
	if err := saveRSAPriv(*out, key); err != nil {
		log.Fatal(err)
	}
	pubOut := *out + ".pub"
	if err := saveRSAPub(pubOut, &key.PublicKey); err != nil {
		log.Fatal(err)
	}
	fmt.Println("generated", *out, "and", pubOut)
}
