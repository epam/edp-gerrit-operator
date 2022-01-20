package gerrit

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestClient_ChangeGet(t *testing.T) {
	cl := Client{
		resty: CreateMockResty(),
	}

	_, err := cl.ChangeGet("foo")
	assert.Error(t, err)

	httpmock.RegisterResponder("GET", "/changes/ch1",
		httpmock.NewStringResponder(200, ")]}' {}"))
	_, err = cl.ChangeGet("ch1")
	assert.NoError(t, err)
}

func TestDecodeGerritResponse(t *testing.T) {
	err := decodeGerritResponse("", nil)
	assert.Error(t, err)
	assert.EqualErrorf(t, err, "wrong gerrit body format", "wrong error")
}

func TestClient_ChangeAbandon(t *testing.T) {
	cl := Client{resty: CreateMockResty()}

	err := cl.ChangeAbandon("foo")
	assert.Error(t, err)

	httpmock.RegisterResponder("POST", "/changes/ch1/abandon",
		httpmock.NewStringResponder(200, ""))
	err = cl.ChangeAbandon("ch1")
	assert.NoError(t, err)
}
