package helpers

import (
	"crypto/ed25519"
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

// LogError prints error message to the log.
func LogError(err error) {
	log.Printf("[ERROR] %v", err)
}

// GeneratePrivateKey generates private key.
func generatePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keyBitSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private rsa key: %w", err)
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate private key: %w", err)
	}

	return privateKey, nil
}

// GeneratePublicKey convert *rsa.PublicKey to ssh.PublicKey.
func generatePublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
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

func GenerateKeyPairs() (privateKey, publicKey []byte, err error) {
	pk, err := generatePrivateKey()
	if err != nil {
		return nil, nil, err
	}

	privateKey = encodePrivateKey(pk)

	publicKey, err = generatePublicKey(pk)
	if err != nil {
		return nil, nil, err
	}

	return
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

// GenerateED25519KeyPairs generates ed25519 key pairs. Private key is in PEM format.
func GenerateED25519KeyPairs() (privateKey, publicKey []byte, err error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}

	asnPrivate, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	pemPrivate := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: asnPrivate,
	}

	sshPublic, err := ssh.NewPublicKey(public)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	return pem.EncodeToMemory(pemPrivate), ssh.MarshalAuthorizedKey(sshPublic), nil
}
