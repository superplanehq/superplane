package files

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type TreeCommand struct{}

func NewTreeCommand() *TreeCommand {
	return &TreeCommand{}
}

func (c *TreeCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("tree accepts at most one positional argument")
	}

	canvasTarget := ""
	if len(ctx.Args) == 1 {
		canvasTarget = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	paths, err := c.listFiles(ctx, canvasID)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"canvasId": canvasID,
			"paths":    paths,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(paths) == 0 {
			_, err := fmt.Fprintln(stdout, "No files found.")
			return err
		}

		return NewFileTree(paths).Write(stdout)
	})
}

func (c *TreeCommand) listFiles(ctx core.CommandContext, canvasID string) ([]string, error) {
	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesListCanvasRepositoryFiles(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(response.GetFiles()))
	for _, file := range response.GetFiles() {
		normalized := file.GetPath()
		if normalized == "" {
			continue
		}
		paths = append(paths, normalized)
	}

	sort.Strings(paths)
	return paths, nil
}

type FileTree struct {
	directories map[string]*FileTree
	files       []string
}

func NewFileTree(paths []string) *FileTree {
	root := &FileTree{
		directories: map[string]*FileTree{},
	}

	for _, path := range paths {
		segments := strings.Split(path, "/")
		node := root
		for index, segment := range segments {
			if segment == "" {
				continue
			}

			isFile := index == len(segments)-1
			if isFile {
				node.files = append(node.files, segment)
				continue
			}

			if node.directories == nil {
				node.directories = map[string]*FileTree{}
			}

			child, ok := node.directories[segment]
			if !ok {
				child = &FileTree{
					directories: map[string]*FileTree{},
				}
				node.directories[segment] = child
			}
			node = child
		}
	}

	root.Sort()
	return root
}

func (t *FileTree) Sort() {
	sort.Strings(t.files)
	if len(t.directories) == 0 {
		return
	}

	names := make([]string, 0, len(t.directories))
	for name := range t.directories {
		names = append(names, name)
	}
	sort.Strings(names)

	sortedDirectories := make(map[string]*FileTree, len(names))
	for _, name := range names {
		child := t.directories[name]
		child.Sort()
		sortedDirectories[name] = child
	}
	t.directories = sortedDirectories
}

type FileTreeEntry struct {
	name  string
	child *FileTree
}

func (t *FileTree) Entries() []FileTreeEntry {
	entries := make([]FileTreeEntry, 0, len(t.directories)+len(t.files))

	for name, child := range t.directories {
		entries = append(entries, FileTreeEntry{
			name:  name + "/",
			child: child,
		})
	}

	for _, name := range t.files {
		entries = append(entries, FileTreeEntry{name: name})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	return entries
}

func (t *FileTree) Write(stdout io.Writer) error {
	entries := t.Entries()
	if len(entries) == 0 {
		return nil
	}

	for index, entry := range entries {
		prefix := "├── "
		if index == len(entries)-1 {
			prefix = "└── "
		}

		if _, err := fmt.Fprintf(stdout, "%s%s\n", prefix, entry.name); err != nil {
			return err
		}

		if entry.child == nil {
			continue
		}

		childPrefix := "│   "
		if index == len(entries)-1 {
			childPrefix = "    "
		}

		if err := entry.child.WriteWithPrefix(stdout, childPrefix); err != nil {
			return err
		}
	}

	return nil
}

func (t *FileTree) WriteWithPrefix(stdout io.Writer, prefix string) error {
	entries := t.Entries()
	for index, entry := range entries {
		branch := "├── "
		nextPrefix := prefix + "│   "
		if index == len(entries)-1 {
			branch = "└── "
			nextPrefix = prefix + "    "
		}

		if _, err := fmt.Fprintf(stdout, "%s%s%s\n", prefix, branch, entry.name); err != nil {
			return err
		}

		if entry.child == nil {
			continue
		}

		if err := entry.child.WriteWithPrefix(stdout, nextPrefix); err != nil {
			return err
		}
	}

	return nil
}
