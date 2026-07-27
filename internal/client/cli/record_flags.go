package cli

import urfavecli "github.com/urfave/cli/v3"

const (
	recordTitleUsage        = "record title"
	recordMetadataFileUsage = "path to optional file with private metadata"
)

func recordMutationFlags(
	withRevision bool,
	titleUsage string,
	metadataUsage string,
	payloadFlags ...urfavecli.Flag,
) []urfavecli.Flag {
	flags := make([]urfavecli.Flag, 0, 3+len(payloadFlags))
	if withRevision {
		flags = append(flags, expectedRevisionFlag())
	}

	flags = append(flags, &urfavecli.StringFlag{
		Name:     titleFlag,
		Usage:    titleUsage,
		Required: true,
	})
	flags = append(flags, payloadFlags...)
	flags = append(flags, &urfavecli.StringFlag{
		Name:  metadataFileFlag,
		Usage: metadataUsage,
	})

	return flags
}

func expectedRevisionFlag() *urfavecli.Int64Flag {
	return &urfavecli.Int64Flag{
		Name:     revisionFlag,
		Aliases:  []string{"r"},
		Usage:    "expected record revision",
		Required: true,
	}
}
