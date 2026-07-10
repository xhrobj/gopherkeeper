package cli

import (
	"context"
	"io"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

type outputRunner func(context.Context, config.Config, io.Writer) error

type textRecordCreateRunner func(
	context.Context,
	config.Config,
	io.Writer,
	string,
	string,
	string,
) error

type textRecordUpdateRunner func(
	context.Context,
	config.Config,
	io.Writer,
	textRecordUpdateCommandRequest,
) error

type textRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	textFile         string
	metadataFile     string
}

type recordGetRunner func(context.Context, config.Config, io.Writer, string) error

type recordDeleteRunner func(context.Context, config.Config, io.Writer, string, int64) error

type passwordRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	string,
	bool,
) error
