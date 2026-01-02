# Pull Request Process

This guide covers the process for submitting and reviewing pull requests.

## Before Submitting

- [ ] Database migrations are created if needed
- [ ] Protobuf files are regenerated if changed
- [ ] Documentation is updated if needed

**Note**: CI will automatically check formatting, linting, builds, and tests. You don't need to run these manually before submitting.

## Commit Sign-Off (DCO)

All commits must be signed off. This certifies that you have the right to contribute the code.

See [Commit Sign-Off Guide](commit_sign-off.md) for complete details.

Quick setup: Add to your `~/.gitconfig`:
```ini
[alias]
  ci = commit -s
```
Then use `git ci` instead of `git commit`.

## Pull Request Title Format

PR titles must follow semantic pull request format. See [Pull Request Titles Guide](pull_request_titles.md) for complete guidelines.

**Format**: `<type>: <description>`

**Types**:
- `feat`: New user-facing features
- `fix`: Bug fixes
- `chore`: Non-user-facing changes (maintenance, tests, refactoring)
- `docs`: Documentation-only changes

**Breaking changes**: Add `!` after the type (e.g., `feat!: Remove deprecated API`)

**Examples**:
- `feat: Add approvals page filters`
- `fix: Handle missing canvas id in logs`
- `chore: Bump Go toolchain version`

## PR Description

Include:
- What changes were made and why
- How to test the changes
- Any breaking changes (if applicable)
- Screenshots for UI changes

## Review Process

- All PRs require at least one approval
- CI checks must pass (tests, linting, DCO verification)
- Address review feedback promptly
- Keep PRs focused and reasonably sized

## Related Guides

- [Commit Sign-Off](commit_sign-off.md) - DCO requirements
- [Pull Request Titles](pull_request_titles.md) - Title format requirements
- [Development Workflow](development-workflow.md) - Making changes

