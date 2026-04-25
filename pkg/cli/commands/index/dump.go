package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"golang.org/x/sync/errgroup"
)

func newDumpCommand(options core.BindOptions) *cobra.Command {
	defaultFile := filepath.Join(os.TempDir(), "superplane-index.json")
	var file string

	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Download full registry (integrations, components, triggers) to a JSON file",
	}
	cmd.Flags().StringVar(&file, "file", defaultFile, "path to write the registry JSON to")
	core.Bind(cmd, &dumpCommand{file: &file}, options)

	return cmd
}

type indexDump struct {
	Integrations []openapi_client.IntegrationsIntegrationDefinition `json:"integrations"`
	Actions      []openapi_client.SuperplaneActionsAction           `json:"actions"`
	Triggers     []openapi_client.TriggersTrigger                   `json:"triggers"`
}

type dumpCommand struct {
	file *string
}

func (c *dumpCommand) Execute(ctx core.CommandContext) error {
	var (
		mu   sync.Mutex
		dump indexDump
	)

	g, gctx := errgroup.WithContext(ctx.Context)

	g.Go(func() error {
		resp, _, err := ctx.API.IntegrationAPI.IntegrationsListIntegrations(gctx).Execute()
		if err != nil {
			return fmt.Errorf("fetching integrations: %w", err)
		}
		mu.Lock()
		dump.Integrations = resp.GetIntegrations()
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		resp, _, err := ctx.API.ActionAPI.ActionsListActions(gctx).Execute()
		if err != nil {
			return fmt.Errorf("fetching actions: %w", err)
		}
		mu.Lock()
		dump.Actions = resp.GetActions()
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		resp, _, err := ctx.API.TriggerAPI.TriggersListTriggers(gctx).Execute()
		if err != nil {
			return fmt.Errorf("fetching triggers: %w", err)
		}
		mu.Lock()
		dump.Triggers = resp.GetTriggers()
		mu.Unlock()
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing registry: %w", err)
	}

	path := *c.file
	if err := writeFileAtomic(path, payload); err != nil {
		return fmt.Errorf("writing to %s: %w", path, err)
	}

	_, err = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Index downloaded to %s\n", path)
	return err
}

// writeFileAtomic writes data to path via a temp file + rename so the target
// is never left in a partially-written state if the process is interrupted.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".superplane-index-*.json")
	if err != nil {
		return fmt.Errorf("cannot create file in %s: %w", dir, unwrapPathError(err))
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("cannot write: %w", unwrapPathError(err))
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("cannot close: %w", unwrapPathError(err))
	}

	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("cannot set permissions: %w", unwrapPathError(err))
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("cannot move to %s: %w", path, unwrapPathError(err))
	}

	return nil
}

// unwrapPathError strips the file path from *os.PathError and *os.LinkError
// so callers see only the OS-level message (e.g. "no such file or directory")
// without internal temp-file names leaking into user-facing output.
func unwrapPathError(err error) error {
	type pathErr interface{ Unwrap() error }
	if pe, ok := err.(pathErr); ok {
		if inner := pe.Unwrap(); inner != nil {
			return inner
		}
	}
	return err
}
