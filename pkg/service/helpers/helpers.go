package helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
)

const (
	keyBitSize = 4096
)

// LogErrorAndReturn prints error message to the log and returns err parameter
func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

// GenerateLabels returns initialized map using name parameter
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

// SaveKeyToFile saves key to file
func SaveKeyToFile(keyBytes []byte, filename string, savePath string) error {
	f, err := os.Create(savePath+filename)
	if err != nil {
		return err
	}
	defer f.Close()

	err = ioutil.WriteFile(savePath+filename, keyBytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

// GeneratePrivateKey generates private key
func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keyBitSize)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// GeneratePublicKey convert *rsa.PublicKey to ssh.PublicKey
func GeneratePublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return publicKeyBytes, nil
}

// EncodePrivateKey encodes private key to PEM format
func EncodePrivateKey(privateKey *rsa.PrivateKey) []byte {
	asnDEREncoding := x509.MarshalPKCS1PrivateKey(privateKey)

	block := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   asnDEREncoding,
	}

	return pem.EncodeToMemory(&block)
}
