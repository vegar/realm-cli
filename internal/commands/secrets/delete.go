package secrets

import (
	"fmt"
	"sort"

	"github.com/10gen/realm-cli/internal/cli"
	"github.com/10gen/realm-cli/internal/cli/user"
	"github.com/10gen/realm-cli/internal/terminal"

	"github.com/spf13/pflag"
)

// CommandMetaDelete is the command meta
var CommandMetaDelete = cli.CommandMeta{
	Use:         "delete",
	Display:     "secrets delete",
	Description: "Delete a Secret from your Realm app",
	HelpText: `With this command, you can:
  - Remove multiple Secrets at once with "--secret" flags. You can specify these
    Secrets using their ID or Name values`,
}

// CommandDelete for the secrets delete command
type CommandDelete struct {
	inputs deleteInputs
}

// Flags function for the secrets delete command
func (cmd *CommandDelete) Flags(fs *pflag.FlagSet) {
	cmd.inputs.Flags(fs)
	fs.StringSliceVarP(&cmd.inputs.secrets, flagSecret, flagSecretShort, []string{}, flagSecretUsageDelete)
}

// Inputs function for the secrets delete command
func (cmd *CommandDelete) Inputs() cli.InputResolver {
	return &cmd.inputs
}

// Handler function for the secrets delete command
func (cmd *CommandDelete) Handler(profile *user.Profile, ui terminal.UI, clients cli.Clients) error {
	app, err := cli.ResolveApp(ui, clients.Realm, cmd.inputs.Filter())
	if err != nil {
		return err
	}

	secrets, err := clients.Realm.Secrets(app.GroupID, app.ID)
	if err != nil {
		return err
	}

	selected, err := cmd.inputs.resolveSecrets(ui, secrets)
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		ui.Print(terminal.NewTextLog("No secrets to delete"))
		return nil
	}

	outputs := make(secretOutputs, len(selected))
	for i, secret := range selected {
		err := clients.Realm.DeleteSecret(app.GroupID, app.ID, secret.ID)
		outputs[i] = secretOutput{secret, err}
	}

	sort.SliceStable(outputs, func(i, j int) bool {
		return outputs[i].err != nil && outputs[j].err == nil
	})

	ui.Print(terminal.NewTableLog(
		fmt.Sprintf("Deleted %d secret(s)", len(outputs)),
		tableHeaders(headerDeleted, headerDetails),
		tableRows(outputs, tableRowDelete)...,
	))
	return nil
}

func tableRowDelete(output secretOutput, row map[string]interface{}) {
	deleted := false
	if output.err != nil {
		row[headerDetails] = output.err.Error()
	} else {
		deleted = true
	}
	row[headerDeleted] = deleted
}
