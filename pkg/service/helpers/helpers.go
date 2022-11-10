package helpers

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	mathrand "math/rand"

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

// GenerateSSHED25519KeyPairs generates ed25519 key pairs. Private key is in PEM format.
func GenerateSSHED25519KeyPairs() (privateKey, publicKey []byte, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}

	sshPublicKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ssh public key: %w", err)
	}

	pemKey := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: marshalED25519PrivateKey(privKey),
	}
	pemPrivateKey := pem.EncodeToMemory(pemKey)
	authorizedKey := ssh.MarshalAuthorizedKey(sshPublicKey)

	return pemPrivateKey, authorizedKey, nil
}

// marshalED25519PrivateKey writes ed25519 private keys into the OpenSSH private key format.
// The x509 package does not support marshaling ed25519 key types in the format used by openssh.
// It is taken from https://github.com/mikesmitty/edkey/blob/master/edkey.go.
// See related topic https://stackoverflow.com/questions/71850135/generate-ed25519-key-pair-compatible-with-openssh.
// nolint:all Disabled all linters for this function to make it the same as in GitHub repository.
func marshalED25519PrivateKey(key ed25519.PrivateKey) []byte {
	// Add our key header (followed by a null byte)
	magic := append([]byte("openssh-key-v1"), 0)

	var w struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}

	// Fill out the private key fields
	pk1 := struct {
		Check1  uint32
		Check2  uint32
		Keytype string
		Pub     []byte
		Priv    []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{}

	// Set our check ints
	ci := mathrand.Uint32()
	pk1.Check1 = ci
	pk1.Check2 = ci
	// Set our key type
	pk1.Keytype = ssh.KeyAlgoED25519

	// Add the pubkey to the optionally-encrypted block
	pk, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		return nil
	}

	pubKey := []byte(pk)
	pk1.Pub = pubKey

	// Add our private key
	pk1.Priv = []byte(key)

	// Might be useful to put something in here at some point
	pk1.Comment = ""

	// Add some padding to match the encryption block size within PrivKeyBlock (without Pad field)
	// 8 doesn't match the documentation, but that's what ssh-keygen uses for unencrypted keys. *shrug*
	bs := 8
	blockLen := len(ssh.Marshal(pk1))
	padLen := (bs - (blockLen % bs)) % bs
	pk1.Pad = make([]byte, padLen)

	// Padding is a sequence of bytes like: 1, 2, 3...
	for i := 0; i < padLen; i++ {
		pk1.Pad[i] = byte(i + 1)
	}

	// Generate the pubkey prefix "\0\0\0\nssh-ed25519\0\0\0 "
	prefix := []byte{0x0, 0x0, 0x0, 0x0b}
	prefix = append(prefix, []byte(ssh.KeyAlgoED25519)...)
	prefix = append(prefix, []byte{0x0, 0x0, 0x0, 0x20}...)

	// Only going to support unencrypted keys for now
	w.CipherName = "none"
	w.KdfName = "none"
	w.KdfOpts = ""
	w.NumKeys = 1
	w.PubKey = append(prefix, pubKey...)
	w.PrivKeyBlock = ssh.Marshal(pk1)

	magic = append(magic, ssh.Marshal(w)...)

	return magic
}
