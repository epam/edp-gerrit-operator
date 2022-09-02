package gerrit

import (
	"fmt"

	"github.com/pkg/errors"
)

type Change struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (gc *Client) ChangeAbandon(changeID string) error {
	rsp, err := gc.resty.R().Post(fmt.Sprintf("changes/%s/abandon", changeID))
	if err = parseRestyResponse(rsp, err); err != nil {
		return errors.Wrap(err, "unable to abandon change")
	}

	return nil
}

func (gc *Client) ChangeGet(changeID string) (*Change, error) {
	rsp, err := gc.resty.R().Get(fmt.Sprintf("changes/%s", changeID))
	if err = parseRestyResponse(rsp, err); err != nil {
		return nil, errors.Wrap(err, "unable to get change")
	}

	var change Change
	if err := decodeGerritResponse(rsp.String(), &change); err != nil {
		return nil, errors.Wrap(err, "unable to decode change from body")
	}

	return &change, nil
}
