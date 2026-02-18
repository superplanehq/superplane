package jsruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

func TestLoadComponents_LoadsValidFiles(t *testing.T) {
	dir := t.TempDir()

	writeJSFile(t, dir, "transform.js", `
		superplane.component({
			label: "Transform",
			description: "Transforms data",
			execute(ctx) {
				ctx.emit("default", "transform.result", {});
			},
		});
	`)

	writeJSFile(t, dir, "validate.js", `
		superplane.component({
			label: "Validate",
			description: "Validates data",
			execute(ctx) {
				ctx.pass();
			},
		});
	`)

	rt := NewRuntime(0)
	reg := newTestRegistry(t)

	loaded, err := LoadComponents(dir, rt, reg)

	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Contains(t, loaded, "js.transform")
	assert.Contains(t, loaded, "js.validate")

	comp, err := reg.GetComponent("js.transform")
	require.NoError(t, err)
	assert.Equal(t, "js.transform", comp.Name())
	assert.Equal(t, "Transform", comp.Label())

	comp, err = reg.GetComponent("js.validate")
	require.NoError(t, err)
	assert.Equal(t, "js.validate", comp.Name())
}

func TestLoadComponents_SkipsInvalidFiles(t *testing.T) {
	dir := t.TempDir()

	writeJSFile(t, dir, "good.js", `
		superplane.component({
			label: "Good",
			description: "Works",
			execute(ctx) { ctx.pass(); },
		});
	`)

	writeJSFile(t, dir, "bad.js", `this is not valid javascript {{{{`)

	rt := NewRuntime(0)
	reg := newTestRegistry(t)

	loaded, err := LoadComponents(dir, rt, reg)

	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Contains(t, loaded, "js.good")
}

func TestLoadComponents_SkipsNonJSFiles(t *testing.T) {
	dir := t.TempDir()

	writeJSFile(t, dir, "transform.js", `
		superplane.component({
			label: "Transform",
			description: "Works",
			execute(ctx) { ctx.pass(); },
		});
	`)

	writeFile(t, dir, "readme.md", "# Not a component")
	writeFile(t, dir, "config.json", "{}")

	rt := NewRuntime(0)
	reg := newTestRegistry(t)

	loaded, err := LoadComponents(dir, rt, reg)

	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}

func TestLoadComponents_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()

	writeJSFile(t, dir, "transform.js", `
		superplane.component({
			label: "Transform",
			description: "Works",
			execute(ctx) { ctx.pass(); },
		});
	`)

	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	rt := NewRuntime(0)
	reg := newTestRegistry(t)

	loaded, err := LoadComponents(dir, rt, reg)

	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}

func TestLoadComponents_InvalidFilename(t *testing.T) {
	dir := t.TempDir()

	writeJSFile(t, dir, "My Component.js", `
		superplane.component({
			label: "Bad Name",
			description: "Invalid filename",
			execute(ctx) { ctx.pass(); },
		});
	`)

	rt := NewRuntime(0)
	reg := newTestRegistry(t)

	loaded, err := LoadComponents(dir, rt, reg)

	require.NoError(t, err)
	assert.Len(t, loaded, 0)
}

func TestRegistryNameFromFilename(t *testing.T) {
	assert.Equal(t, "js.transform", RegistryNameFromFilename("transform.js"))
	assert.Equal(t, "js.slack-notify", RegistryNameFromFilename("slack-notify.js"))
}

func writeJSFile(t *testing.T, dir, name, content string) {
	t.Helper()
	writeFile(t, dir, name, content)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}

func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()

	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	require.NoError(t, err)

	return reg
}
