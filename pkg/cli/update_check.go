package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
)

const (
	// ConfigKeyLastUpdateCheck holds the time of the last update check in the
	// CLI configuration file. Viper writes configuration keys in lower case, so
	// the key is stored in lower case and read without case sensitivity.
	ConfigKeyLastUpdateCheck = "lastupdatecheck"

	// updateCheckInterval is the minimum time between two update checks.
	updateCheckInterval = 24 * time.Hour

	// configFileName is the name of the CLI configuration file in the home
	// directory.
	configFileName = ".superplane.yaml"

	// newConfigFileMode applies only when the configuration file does not exist
	// yet. The file can hold an API token, so it stays private to the user.
	newConfigFileMode = os.FileMode(0600)
)

// timeNow gives the current time. Tests replace it to control the clock.
var timeNow = time.Now

// configFilePath gives the configuration file that the CLI reads for the given
// command line arguments. It repeats the resolution in initConfig, because the
// update check runs before Cobra parses the flags.
func configFilePath(args []string) string {
	if path := configFlagValue(args); path != "" {
		return path
	}

	home, err := homedir.Dir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, configFileName)
}

func configFlagValue(args []string) string {
	for i, arg := range args {
		if value, found := strings.CutPrefix(arg, "--config="); found {
			return value
		}

		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
	}

	return ""
}

// updateCheckDue reports whether the CLI must look for a new release. A
// configuration file that does not exist, or that has no readable timestamp,
// counts as "never checked".
func updateCheckDue(path string) bool {
	last, found := lastUpdateCheck(path)
	if !found {
		return true
	}

	return timeNow().Sub(last) >= updateCheckInterval
}

func lastUpdateCheck(path string) (time.Time, bool) {
	document, err := readConfigDocument(path)
	if err != nil {
		return time.Time{}, false
	}

	_, value, found := findEntry(mappingOf(document), ConfigKeyLastUpdateCheck)
	if !found {
		return time.Time{}, false
	}

	last, err := time.Parse(time.RFC3339, value.Value)
	if err != nil {
		return time.Time{}, false
	}

	return last, true
}

// recordUpdateCheck writes the current time to the configuration file. It edits
// the YAML document in place, so that the comments, the key order and the
// indentation of the file stay as the user has them. The file also holds the
// contexts and the API token, and this write runs without the user asking for
// it.
func recordUpdateCheck(path string) error {
	document, err := readConfigDocument(path)
	if err != nil {
		return err
	}

	setEntry(mappingOf(document), ConfigKeyLastUpdateCheck, timeNow().UTC().Format(time.RFC3339))

	contents, err := yaml.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to encode configuration: %w", err)
	}

	if err := os.WriteFile(path, contents, configFileMode(path)); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	return nil
}

// readConfigDocument decodes the configuration file. A file that is absent or
// empty gives an empty document, so that the first run of the CLI can record a
// timestamp.
func readConfigDocument(path string) (*yaml.Node, error) {
	if path == "" {
		return nil, fmt.Errorf("no configuration file path")
	}

	contents, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(contents, &document); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(document.Content) == 0 {
		return emptyDocument(), nil
	}

	if document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("configuration is not a YAML mapping")
	}

	return &document, nil
}

func emptyDocument() *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
	}
}

func mappingOf(document *yaml.Node) *yaml.Node {
	return document.Content[0]
}

// findEntry looks up a key without case sensitivity, because Viper writes
// configuration keys in lower case and a user can edit the file by hand.
func findEntry(mapping *yaml.Node, key string) (*yaml.Node, *yaml.Node, bool) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if strings.EqualFold(mapping.Content[i].Value, key) {
			return mapping.Content[i], mapping.Content[i+1], true
		}
	}

	return nil, nil, false
}

// setEntry updates the value of a key that exists, and keeps the name that the
// file uses. It appends the key when the file does not have it.
func setEntry(mapping *yaml.Node, key, value string) {
	if _, current, found := findEntry(mapping, key); found {
		current.SetString(value)
		return
	}

	mapping.Content = append(
		mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

// configFileMode keeps the mode of a configuration file that exists, and uses a
// private mode for a file that the CLI creates.
func configFileMode(path string) os.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return newConfigFileMode
	}

	return info.Mode().Perm()
}
