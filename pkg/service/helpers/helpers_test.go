package helpers

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/epam/edp-gerrit-operator/v2/pkg/service/gerrit/spec"
)

func TestLogErrorAndReturn(t *testing.T) {
	err := errors.New("test")
	assert.Equal(t, err, LogErrorAndReturn(err))
}

func Test_generatePublicKey(t *testing.T) {
	private, err := generatePrivateKey()
	require.NoError(t, err)

	publicKey, err := ssh.NewPublicKey(&private.PublicKey)
	require.NoError(t, err)

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	key, err := generatePublicKey(private)

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
	require.NoError(t, err)

	decode, _ := pem.Decode(private)

	key, err := x509.ParsePKCS1PrivateKey(decode.Bytes)
	assert.NoError(t, err)

	publicKey, err := generatePublicKey(key)
	assert.NoError(t, err)
	assert.Equal(t, publicKey, public)
}
