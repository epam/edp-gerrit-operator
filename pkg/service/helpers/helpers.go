package helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"

	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
)

const (
	keyBitSize = 4096
)

// LogErrorAndReturn prints error message to the log and returns err parameter.
func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

// GeneratePrivateKey generates private key.
func generatePrivateKey() (*rsa.PrivateKey, error) {
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

// GeneratePublicKey convert *rsa.PublicKey to ssh.PublicKey.
func generatePublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return publicKeyBytes, nil
}

// EncodePrivateKey encodes private key to PEM format.
func encodePrivateKey(privateKey *rsa.PrivateKey) []byte {
	asnDEREncoding := x509.MarshalPKCS1PrivateKey(privateKey)

	block := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   asnDEREncoding,
	}

	return pem.EncodeToMemory(&block)
}

func GenerateKeyPairs() ([]byte, []byte, error) {
	privateKey, err := generatePrivateKey()
	if err != nil {
		return nil, nil, err
	}

	publicKey, err := generatePublicKey(privateKey)
	if err != nil {
		return nil, nil, err
	}

	return encodePrivateKey(privateKey), publicKey, nil
}

func IsStringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func GenerateAnnotationKey(entitySuffix string) string {
	return fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, entitySuffix)
}
