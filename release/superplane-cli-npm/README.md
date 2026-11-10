# @superplane/cli

The SuperPlane CLI, distributed via npm.

```sh
npm install -g @superplane/cli
superplane --help
```

On install, a small `postinstall` step downloads the matching platform binary
from `https://install.superplane.com` and places it inside the package.
The package itself is a thin Node wrapper that execs that binary.

## Supported platforms

| OS     | Architecture |
| ------ | ------------ |
| macOS  | x64, arm64   |
| Linux  | x64, arm64   |

Windows is not yet supported through this package.

## Alternatives

- Direct download: `curl -fsSL https://install.superplane.com/install.sh | sh`

Public APT publishing is retired. Use npm or direct download on Debian and Ubuntu.

## Links

- Documentation: https://docs.superplane.com
- Source: https://github.com/superplanehq/superplane
- Issues: https://github.com/superplanehq/superplane/issues
