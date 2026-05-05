# CLI Release Assets

`release/cli/` is the staging directory used by `release/create-github-release.js` for CLI release uploads.

Before running the release script, this directory must contain:

- `superplane-cli-darwin-arm64`
- `superplane-cli-darwin-amd64`
- `superplane-cli-linux-arm64`
- `superplane-cli-linux-amd64`
- `install.sh`

The release script uploads all staged `superplane-cli-*` binaries and `install.sh`.
