package gerrit

import (
	"testing"

	"github.com/pkg/errors"
)

func TestIsErrUserNotFound(t *testing.T) {
	tnf := ErrUserNotFound("err not found")
	err := errors.Wrap(tnf, "error")
	if !IsErrUserNotFound(err) {
		t.Fatal("wrong error type")
	}
}
