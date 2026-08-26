package cli

import (
	"github.com/chaoscondensate/cli/internal/presentation"
	urfavecli "github.com/urfave/cli/v3"
)

func presenterFor(command *urfavecli.Command) *presentation.Presenter {
	root := command.Root()
	return presentation.New(root.Writer, root.ErrWriter, presentation.Options{
		JSON: root.Bool("json"), Plain: root.Bool("plain"), Quiet: root.Bool("quiet"), Verbose: root.Bool("verbose"), NoColor: root.Bool("no-color"),
	})
}
