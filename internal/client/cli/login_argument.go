package cli

import (
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var errInvalidLoginArgument = errors.New(
	"invalid login: must be 3-32 characters, start with an ASCII letter or digit, " +
		"and contain only ASCII letters, digits, '.', '_' or '-'",
)

func canonicalizeLoginArgument(login string) (string, error) {
	canonicalLogin, err := model.CanonicalizeLogin(login)
	if err != nil {
		return "", errInvalidLoginArgument
	}

	return canonicalLogin, nil
}
