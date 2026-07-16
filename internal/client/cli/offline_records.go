package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

const (
	offlineFlag = "offline"
	loginFlag   = "login"
)

var (
	errOfflineLoginRequired = errors.New("login is required in offline mode")
	errLoginRequiresOffline = errors.New("--login can only be used with --offline")
)

type recordReadMode struct {
	offline bool
	login   string
}

func offlineReadFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.BoolFlag{
			Name:  offlineFlag,
			Usage: "read from the encrypted local cache",
		},
		&urfavecli.StringFlag{
			Name:    loginFlag,
			Aliases: []string{"l"},
			Usage:   "account login for the encrypted local cache",
		},
	}
}

func recordReadModeFromCommand(command *urfavecli.Command) (recordReadMode, error) {
	offline := command.Bool(offlineFlag)
	loginSet := command.IsSet(loginFlag)

	if !offline {
		if loginSet {
			return recordReadMode{}, errLoginRequiresOffline
		}

		return recordReadMode{}, nil
	}

	login := command.String(loginFlag)
	if !loginSet || strings.TrimSpace(login) == "" {
		return recordReadMode{}, errOfflineLoginRequired
	}

	canonicalLogin, err := canonicalizeLoginArgument(login)
	if err != nil {
		return recordReadMode{}, err
	}

	return recordReadMode{offline: true, login: canonicalLogin}, nil
}

func executeOfflineListRecords(
	ctx context.Context,
	application application,
	passwords passwordReader,
	streams passwordStreams,
	login string,
) error {
	password, err := passwords.ReadHidden(streams.input, streams.promptOutput, "Password: ")
	if err != nil {
		return err
	}

	result, err := application.ListCachedRecords(ctx, usecase.OfflineReadRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return err
	}

	return writeOfflineRecordList(streams.output, result)
}

func executeOfflineGetRecord(
	ctx context.Context,
	application application,
	passwords passwordReader,
	streams passwordStreams,
	login string,
	recordID string,
	outputPath string,
) error {
	password, err := passwords.ReadHidden(streams.input, streams.promptOutput, "Password: ")
	if err != nil {
		return err
	}

	result, err := application.GetCachedRecord(
		ctx,
		usecase.OfflineReadRequest{Login: login, Password: password},
		recordID,
	)
	if err != nil {
		return err
	}

	return writeOfflineRecord(streams.output, result, outputPath)
}

func writeOfflineRecordList(output io.Writer, result usecase.OfflineListResult) error {
	var body bytes.Buffer
	if len(result.Records) == 0 {
		if _, err := fmt.Fprintln(&body, "No cached records found."); err != nil {
			return fmt.Errorf("prepare empty cached record list: %w", err)
		}
	} else if err := writeRecordList(&body, result.Records); err != nil {
		return err
	}

	return writeOfflineResult(output, result.Source, result.MayBeStale, &body)
}

func writeOfflineRecord(output io.Writer, result usecase.OfflineGetResult, outputPath string) error {
	var body bytes.Buffer
	if err := writeRecord(&body, result.Record, outputPath); err != nil {
		return err
	}

	return writeOfflineResult(output, result.Source, result.MayBeStale, &body)
}

func writeOfflineResult(
	output io.Writer,
	source usecase.OfflineSource,
	mayBeStale bool,
	body io.Reader,
) error {
	sourceLabel := string(source)
	if source == usecase.OfflineSourceLocalCache {
		sourceLabel = "encrypted local cache"
	}

	message := fmt.Sprintf("Source: %s.", sourceLabel)
	if mayBeStale {
		message = fmt.Sprintf("Source: %s (data may be stale).", sourceLabel)
	}

	if _, err := fmt.Fprintln(output, message); err != nil {
		return fmt.Errorf("write offline source: %w", err)
	}
	if _, err := io.Copy(output, body); err != nil {
		return fmt.Errorf("write offline result: %w", err)
	}

	return nil
}
