package helper

import (
	"errors"
	"testing"
)

func TestInfoLogger_All(t *testing.T) {
	m := Logger{}
	m1 := m.V(1).WithValues("fo", "bar").WithName("n")
	if !m1.Enabled() {
		t.Fatal("not enabled")
	}

	m1.Info("i", "fo", "ba")
	m1.Error(errors.New("fo"), "msg")
	if m1.(*Logger).LastError().Error() != "fo" {
		t.Fatal("wrong error")
	}
}
