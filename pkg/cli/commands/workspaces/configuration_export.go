package workspaces

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type configurationExportCommand struct {
	configurationTarget
	outputDir string
	overwrite bool
}

func newConfigurationExportCommand(options core.BindOptions) *cobra.Command {
	command := &configurationExportCommand{}
	cmd := &cobra.Command{
		Use: "export", Short: "Export repositories, stages, and automations",
		Long: "Export workspace repositories, stages, and automations. Repository settings are informational. Change repositories in workspace settings and export again.", Args: cobra.NoArgs,
	}
	command.bind(cmd)
	cmd.Flags().StringVar(&command.outputDir, "output-dir", ".", "destination directory for the .superplane folder")
	cmd.Flags().BoolVar(&command.overwrite, "overwrite", false, "replace existing exported configuration files")
	core.Bind(cmd, command, options)
	return cmd
}

func (c *configurationExportCommand) Execute(ctx core.CommandContext) error {
	if err := c.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.outputDir) == "" {
		return fmt.Errorf("pass --output-dir with a destination directory")
	}
	response, _, err := ctx.API.FactoryAPI.
		FactoriesExportFactoryConfiguration(ctx.Context, strings.TrimSpace(c.workspace)).Execute()
	if err != nil {
		return err
	}
	files, err := validateConfigurationExport(response)
	if err != nil {
		return err
	}
	if err := c.writeFiles(files); err != nil {
		return err
	}
	manifestPath := filepath.Join(c.outputDir, filepath.FromSlash(configurationManifestPath))
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]string{"coverage": response.GetCoverage(), "manifest_path": manifestPath})
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Exported %s to %s (partial workspace configuration).\n", response.GetCoverage(), manifestPath)
		return err
	})
}

func validateConfigurationExport(response *openapi_client.FactoriesExportFactoryConfigurationResponse) ([]openapi_client.FactoriesFactoryConfigurationFile, error) {
	if response == nil || response.GetCoverage() != configurationCoverage || !response.HasBundle() {
		return nil, fmt.Errorf("invalid export response: expected %s and a bundle", configurationCoverage)
	}
	bundle := response.GetBundle()
	if bundle.GetManifestPath() != configurationManifestPath {
		return nil, fmt.Errorf("invalid export response: expected manifest path %s", configurationManifestPath)
	}
	byPath := map[string]openapi_client.FactoriesFactoryConfigurationFile{}
	for _, file := range bundle.GetFiles() {
		filePath := file.GetPath()
		if filePath != configurationManifestPath && (!strings.HasPrefix(filePath, ".superplane/") || !configurationReferenceAllowed(strings.TrimPrefix(filePath, ".superplane/"))) {
			return nil, fmt.Errorf("invalid export response: unsafe file path %q", filePath)
		}
		if _, exists := byPath[filePath]; exists || !file.HasContent() {
			return nil, fmt.Errorf("invalid export response: duplicate file or missing content for %q", filePath)
		}
		byPath[filePath] = file
	}
	manifest, exists := byPath[configurationManifestPath]
	if !exists {
		return nil, fmt.Errorf("invalid export response: missing manifest content")
	}
	references, err := configurationManifestReferences(manifest.GetContent())
	if err != nil {
		return nil, err
	}
	if len(byPath) != len(references)+1 {
		return nil, fmt.Errorf("invalid export response: files must match the manifest references")
	}
	files := make([]openapi_client.FactoriesFactoryConfigurationFile, 0, len(byPath))
	for _, reference := range references {
		file, exists := byPath[".superplane/"+reference]
		if !exists {
			return nil, fmt.Errorf("invalid export response: missing custom automation %q", reference)
		}
		files = append(files, file)
	}
	// Publish the manifest after every referenced file is ready.
	return append(files, manifest), nil
}

type stagedConfigurationFile struct {
	directory *os.Root
	name      string
	temporary string
}

func (c *configurationExportCommand) writeFiles(files []openapi_client.FactoriesFactoryConfigurationFile) error {
	directory, err := openConfigurationOutputDirectory(c.outputDir)
	if err != nil {
		return err
	}
	defer directory.Close()
	staged := make([]stagedConfigurationFile, 0, len(files))
	defer func() {
		for _, file := range staged {
			if file.temporary != "" {
				_ = file.directory.Remove(file.temporary)
			}
			_ = file.directory.Close()
		}
	}()
	// Check every destination before writing any file content.
	for _, file := range files {
		parent, err := openConfigurationFileDirectory(directory, path.Dir(file.GetPath()))
		if err != nil {
			return err
		}
		staged = append(staged, stagedConfigurationFile{directory: parent, name: path.Base(file.GetPath())})
		info, err := parent.Lstat(path.Base(file.GetPath()))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsafe export path: %s must be a regular file", file.GetPath())
			}
			if !c.overwrite {
				return fmt.Errorf("%s already exists; pass --overwrite to replace it", file.GetPath())
			}
		}
	}
	for index, file := range files {
		staged[index].temporary = ".factory-" + rand.Text() + ".yaml"
		if err := stageConfigurationFile(staged[index], file.GetContent()); err != nil {
			return err
		}
	}
	for _, file := range staged {
		if c.overwrite {
			if err := file.directory.Rename(file.temporary, file.name); err != nil {
				return err
			}
			continue
		}
		// Link publishes a complete file without replacing a concurrent export.
		if err := file.directory.Link(file.temporary, file.name); err != nil {
			if errors.Is(err, os.ErrExist) {
				return fmt.Errorf("%s already exists; pass --overwrite to replace it", file.name)
			}
			return err
		}
	}
	return nil
}

func stageConfigurationFile(file stagedConfigurationFile, content string) error {
	temporary, err := file.directory.OpenFile(file.temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(temporary, content)
	if writeErr == nil {
		writeErr = temporary.Sync()
	}
	closeErr := temporary.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func openConfigurationFileDirectory(directory *os.Root, relative string) (*os.Root, error) {
	parts := strings.Split(relative, "/")
	for index := range parts {
		current := strings.Join(parts[:index+1], "/")
		if err := directory.Mkdir(current, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		info, err := directory.Lstat(current)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("unsafe export directory: %s must be a directory without symbolic links", current)
		}
	}
	return directory.OpenRoot(relative)
}

func openConfigurationOutputDirectory(path string) (*os.Root, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	rootPath := filepath.VolumeName(absolute) + string(filepath.Separator)
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(absolute, rootPath), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		if err := root.Mkdir(part, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			root.Close()
			return nil, err
		}
		info, err := root.Lstat(part)
		if err != nil {
			root.Close()
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, fmt.Errorf("unsafe export directory: %s must be a directory without symbolic links", part)
		}
		next, err := root.OpenRoot(part)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = next
	}
	return root, nil
}
