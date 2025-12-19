# Developer's Certificate of Origin

- [Overview](#overview)
- [Adding an alias command to git that signs off every commit](#adding-an-alias-command-to-git-that-signs-off-every-commit)
- [Automatically sign all commits in VSCode](#automatically-sign-all-commits-in-vscode)
- [Automatic verification of DCO](#automatic-verification-of-dco)
- [Fixing commits that don't have a signed-off message](#fixing-commits-that-dont-have-a-signed-off-message)

## Overview

Any contributions to Superplane must only contain code that can legally be
contributed to Superplane, and which the Superplane project can distribute
under its license.

Prior to contributing to Superplane please read the [Developer's Certificate of Origin](/docs/legal/developer_certificate_of_origin.txt)
and sign-off all commits with the `--signoff` option provided by `git commit`.

For example:

```
git commit --signoff --message "This is the commit message"
```

This will add a `Signed-off-by` trailer to the end of the commit log message.

### Adding an alias command to git that signs off every commit

To automatically signoff every commit, put the following in your `~/.gitconfig`:

```
[user]
  email = <YOUR_EMAIL>
  name = <YOUR FULL NAME>

[alias]
  ci = commit -s
```

Use the new `git ci` alias to make commits: `git ci -m "Example commit messages"`.

### Automatically sign all commits in VSCode

To automatically append the sign-off line to every commit made via the VS Code
interface, you can enable a setting:

1. Open VS Code Settings (Code > Settings on macOS, or File > Preferences > Settings on Linux/Windows)
2. Search for git.enableCommitSigning or git signoff.
3. Check the box for "Git: Enable Commit Signing".

This setting automatically appends the -s or --signoff flag to the git commit
command when you commit via the VS Code UI, achieving the same result as running
git commit -s in the terminal.

### Automatic verification of DCO

When you open a Pull-Request, the [Github DCO App](https://github.com/apps/dco) will automatically
check every commit in your pull request.

### Fixing commits that don't have a signed-off message

If you accidentally forgot to sign-off your commits, you can do one of the following:

- For individual commits, do `git commit --amend --no-edit --signoff`.
- Or for multiple commits, start an interactive rebase with the main branch: `git rebase -i main` and make sure to signoff every commit.
