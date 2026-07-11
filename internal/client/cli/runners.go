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

type credentialsRecordCreateRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	credentialsRecordCreateCommandRequest,
) error

type credentialsRecordUpdateRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	credentialsRecordUpdateCommandRequest,
) error

type credentialsRecordCreateCommandRequest struct {
	title            string
	metadataFile     string
	credentialsStdin bool
}

type credentialsRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	metadataFile     string
	credentialsStdin bool
}

type cardRecordCreateRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	cardRecordCreateCommandRequest,
) error

type cardRecordUpdateRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	cardRecordUpdateCommandRequest,
) error

type cardRecordCreateCommandRequest struct {
	title        string
	metadataFile string
	cardStdin    bool
}

type cardRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	metadataFile     string
	cardStdin        bool
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
