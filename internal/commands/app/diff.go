package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/10gen/realm-cli/internal/cli"
	"github.com/10gen/realm-cli/internal/cli/user"
	"github.com/10gen/realm-cli/internal/cloud/realm"
	"github.com/10gen/realm-cli/internal/local"
	"github.com/10gen/realm-cli/internal/terminal"
	"github.com/10gen/realm-cli/internal/utils/flags"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/pflag"
)

// CommandMetaDiff is the command meta
var CommandMetaDiff = cli.CommandMeta{
	Use:         "diff",
	Aliases:     []string{},
	Display:     "app diff",
	Description: "Show differences between your local directory and your Realm app",
	HelpText: `Displays file-by-file differences between your local directory and the latest
version of your Realm app. If you have more than one Realm app, you will be
prompted to select a Realm app to view.`,
}

// CommandDiff is the `app diff` command
type CommandDiff struct {
	inputs diffInputs
}

type diffInputs struct {
	LocalPath           string
	RemoteApp           string
	Project             string
	IncludeDependencies bool
	IncludeHosting      bool
}

const (
	flagLocalPathDiff      = "local"
	flagLocalPathDiffUsage = "the local path to a Realm app to diff"

	flagRemoteAppDiff      = "remote"
	flagRemoteAppDiffUsage = "a remote Realm app (id or name) to diff changes from"

	flagProjectDiff      = "project"
	flagProjectDiffUsage = "the MongoDB cloud project id"

	flagIncludeDependencies      = "include-dependencies"
	flagIncludeDependenciesShort = "d"
	flagIncludeDependenciesUsage = "include to diff Realm app dependencies changes as well"

	flagIncludeHosting      = "include-hosting"
	flagIncludeHostingShort = "s"
	flagIncludeHostingUsage = "include to diff Realm app hosting changes as well"
)

// Flags is the command flags
func (cmd *CommandDiff) Flags(fs *pflag.FlagSet) {
	fs.StringVar(&cmd.inputs.LocalPath, flagLocalPathDiff, "", flagLocalPathDiffUsage)
	fs.StringVar(&cmd.inputs.RemoteApp, flagRemoteAppDiff, "", flagRemoteAppDiffUsage)
	fs.BoolVarP(&cmd.inputs.IncludeDependencies, flagIncludeDependencies, flagIncludeDependenciesShort, false, flagIncludeDependenciesUsage)
	fs.BoolVarP(&cmd.inputs.IncludeHosting, flagIncludeHosting, flagIncludeHostingShort, false, flagIncludeHostingUsage)

	fs.StringVar(&cmd.inputs.Project, flagProjectDiff, "", flagProjectDiffUsage)
	flags.MarkHidden(fs, flagProjectDiff)
}

// Inputs is the command inputs
func (cmd *CommandDiff) Inputs() cli.InputResolver {
	return &cmd.inputs
}

// Handler is the command handler
func (cmd *CommandDiff) Handler(profile *user.Profile, ui terminal.UI, clients cli.Clients) error {
	app, err := local.LoadApp(cmd.inputs.LocalPath)
	if err != nil {
		return err
	}

	if app.RootDir == "" {
		return fmt.Errorf("no app directory found at %s", cmd.inputs.LocalPath)
	}

	appToDiff, err := cli.ResolveApp(ui, clients.Realm, realm.AppFilter{GroupID: cmd.inputs.Project, App: cmd.inputs.RemoteApp})
	if err != nil {
		return err
	}

	diffs, err := clients.Realm.Diff(appToDiff.GroupID, appToDiff.ID, app.AppData)
	if err != nil {
		return err
	}

	if cmd.inputs.IncludeDependencies {
		uploadPath, err := local.PrepareDependencies(app, ui)
		if err != nil {
			return err
		}
		defer os.Remove(uploadPath) //nolint:errcheck

		dependenciesDiff, err := clients.Realm.DiffDependencies(appToDiff.GroupID, appToDiff.ID, uploadPath)
		if err != nil {
			return err
		}
		diffs = append(diffs, dependenciesDiff.Strings()...)
	}

	if cmd.inputs.IncludeHosting {
		hosting, err := local.FindAppHosting(app.RootDir)
		if err != nil {
			return err
		}

		appAssets, err := clients.Realm.HostingAssets(appToDiff.GroupID, appToDiff.ID)
		if err != nil {
			return err
		}

		hostingDiffs, err := hosting.Diffs(profile.HostingAssetCachePath(), appToDiff.ID, appAssets)
		if err != nil {
			return err
		}

		diffs = append(diffs, hostingDiffs.Strings()...)
	}

	if len(diffs) == 0 {
		// there are no diffs
		ui.Print(terminal.NewTextLog("Deployed app is identical to proposed version"))
		return nil
	}

	ui.Print(terminal.NewTextLog(
		"The following reflects the proposed changes to your Realm app\n%s",
		strings.Join(diffs, "\n"),
	))

	return nil
}

func (i *diffInputs) Resolve(profile *user.Profile, ui terminal.UI) error {
	searchPath := i.LocalPath
	if searchPath == "" {
		searchPath = profile.WorkingDirectory
	}

	app, err := local.LoadAppConfig(searchPath)
	if err != nil {
		return err
	}

	if i.LocalPath == "" && app.RootDir == "" {
		if err := ui.AskOne(&i.LocalPath, &survey.Input{Message: "App filepath (local)"}); err != nil {
			return err
		}

		app, err = local.LoadAppConfig(i.LocalPath)
		if err != nil {
			return err
		}
	}

	if app.RootDir != "" {
		i.LocalPath = app.RootDir
	}

	if i.RemoteApp == "" {
		i.RemoteApp = app.Option()
	}

	return nil
}
