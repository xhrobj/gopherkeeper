package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

type currentUserGetterFunc func(context.Context) (model.User, error)

func (f currentUserGetterFunc) Whoami(ctx context.Context) (model.User, error) {
	return f(ctx)
}

func TestExecuteWhoami(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	var output bytes.Buffer

	err := executeWhoami(
		context.Background(),
		currentUserGetterFunc(func(context.Context) (model.User, error) {
			return model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: createdAt,
			}, nil
		}),
		&output,
	)
	if err != nil {
		t.Fatalf("executeWhoami() error = %v", err)
	}

	if got := output.String(); got != "alice\n" {
		t.Errorf("output = %q, want alice", got)
	}
	if strings.Contains(output.String(), "test.jwt.token") {
		t.Error("whoami output contains access token")
	}
}

func TestExecuteWhoami_ReturnsApplicationError(t *testing.T) {
	applicationError := errors.New("online session is invalid or expired: run gkeep login")

	err := executeWhoami(
		context.Background(),
		currentUserGetterFunc(func(context.Context) (model.User, error) {
			return model.User{}, applicationError
		}),
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("executeWhoami() error = nil, want application error")
	}
	if !errors.Is(err, applicationError) {
		t.Error("whoami error does not preserve application error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("application error contains access token")
	}
}
