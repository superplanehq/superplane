# Development Workflow

This guide covers the day-to-day workflow for making contributions to Superplane.

## Quick Setup

```bash
# Initial setup (first time)
make dev.setup

# Start the application
make dev.start
```

The application will be available at http://localhost:8000

## Making Changes

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes** following the code style guidelines (see [Guidelines](guidelines.md))

3. **Run tests** (optional - CI will run these automatically):
   ```bash
   make test                    # Backend tests
   make test.e2e               # E2E tests (see docs/development/e2e_tests.md)
   ```

4. **Commit with sign-off** (required - see [Commit Sign-Off](commit_sign-off.md))

## Local Checks (Optional)

CI will automatically run formatting checks, linting, and build verification. You can run these locally if you want to catch issues early:

```bash
make format.go              # Format Go code
make format.js              # Format JavaScript/TypeScript code
make lint                   # Lint Go code
make check.build.ui         # Verify UI builds
make check.build.app        # Verify backend builds
```

## Database Migrations

After pulling changes that include new migrations:

```bash
make db.migrate DB_NAME=superplane_dev
```

To create a new migration:

```bash
make db.migration.create NAME=your-migration-name
```

**Important**: Always use dashes instead of underscores in migration names. We do not write rollback migrations, so leave the `*.down.sql` files empty.

## Related Guides

- [Guidelines](guidelines.md) - Code style and development guidelines
- [Pull Requests](pull-requests.md) - Submitting and reviewing PRs
- [Local Development](docs/installation/local-development.md) - Detailed setup instructions

