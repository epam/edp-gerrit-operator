package helpers

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	"testing"
)

func TestLogErrorAndReturn(t *testing.T) {
	err := errors.New("test")
	assert.Equal(t, err, LogErrorAndReturn(err))
}

func Test_generatePublicKey(t *testing.T) {

	priv, err := generatePrivateKey()
	assert.NoError(t, err)
	publicKey, err := ssh.NewPublicKey(&priv.PublicKey)
	assert.NoError(t, err)
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	key, err := generatePublicKey(priv)
	assert.NoError(t, err)
	assert.Equal(t, publicKeyBytes, key)
}

func TestIsStringInSliceTrue(t *testing.T) {
	strs := []string{
		"one", "two",
	}
	str := "one"
	assert.True(t, IsStringInSlice(str, strs))
}

func TestIsStringInSliceFalse(t *testing.T) {
	strs := []string{
		"one", "two",
	}
	str := "nine"
	assert.False(t, IsStringInSlice(str, strs))
}

func TestGenerateAnnotationKey(t *testing.T) {
	str := "test"
	assert.Equal(t, fmt.Sprintf("%v/%v", spec.EdpAnnotationsPrefix, str), GenerateAnnotationKey(str))
}

func TestGenerateKeyPairs(t *testing.T) {
	private, public, err := GenerateKeyPairs()
	assert.NoError(t, err)
	decode, _ := pem.Decode(private)
	key, err := x509.ParsePKCS1PrivateKey(decode.Bytes)
	assert.NoError(t, err)
	publicKey, err := generatePublicKey(key)
	assert.NoError(t, err)
	assert.Equal(t, publicKey, public)
}
