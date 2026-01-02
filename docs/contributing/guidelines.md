# Development Guidelines

This document outlines key guidelines and best practices for contributing to Superplane.

## Code Style

### Go (Backend)

- **Early returns**: Prefer early returns over else blocks
- **Type usage**: Use `any` instead of `interface{}`
- **Slice operations**: Use `slice.Contains` or `slice.ContainsFunc` for checking item existence
- **Variable naming**: Avoid type suffixes like `*Str` or `*UUID`; Go is typed
- **Test timestamps**: Use `time.Now()` as a base instead of absolute `time.Date` times
- **File naming**: Test files end with `_test.go`

### TypeScript/React (Frontend)

- **Strict mode**: All TypeScript types must be properly defined with no implicit `any`
- **Named exports**: Prefer named exports over default exports
- **Component design**: Components should be self-contained and manage their own state
- **No mock data**: Mock data belongs in Storybook stories, not in component files
- **Type inline handlers**: Always type inline handler parameters explicitly

See [UI Development Guide](../../web_src/AGENTS.md) for detailed frontend development patterns.

## Database Guidelines

### Database Transactions

**Critical**: When working with database transactions, follow these rules:

- **NEVER** call `database.Conn()` inside a function that receives a `tx *gorm.DB` parameter
- **Always propagate** the transaction context through the entire call chain
- **Context constructors** must accept `tx *gorm.DB` if they perform database queries
- **When creating new model methods**: Create both variants:
  - `FindUser(id)` - non-transaction variant (uses `database.Conn()`)
  - `FindUserInTransaction(tx *gorm.DB, id)` - transaction variant
  - The non-transaction variant should call the transaction variant with `database.Conn()`

### Database Models

When adding new model methods that need transaction support:
1. Create the transaction variant: `FindUserInTransaction(tx *gorm.DB, id)`
2. Create the non-transaction variant: `FindUser(id)` that calls the transaction variant with `database.Conn()`

### Migrations

- Use dashes, not underscores, in migration names
- Leave `*.down.sql` files empty (we don't write rollbacks)
- After creating a migration, run: `make db.migrate DB_NAME=superplane_dev`

## API & Protobuf Changes

When updating protobuf definitions in `protos/`:

1. **Regenerate protobuf files**:
   ```bash
   make pb.gen
   ```

2. **Generate OpenAPI spec**:
   ```bash
   make openapi.spec.gen
   ```

3. **Generate SDKs**:
   ```bash
   # Go SDK for CLI
   make openapi.client.gen
   
   # TypeScript SDK for UI
   make openapi.web.client.gen
   ```

4. **Validate enum mappings**: Ensure enums are properly mapped to constants in `pkg/models`. Check `Proto*` and `*ToProto` functions in `pkg/grpc/actions/common.go`.

5. **Update authorization**: After adding new API endpoints, ensure authorization is covered in `pkg/authorization/interceptor.go`

## Adding New Features

### Adding a New Worker

1. Add worker code in `pkg/workers`
2. Add startup in `cmd/server/main.go`
3. Update docker-compose files with required environment variables

### Adding New API Endpoints

1. Define in protobuf (`protos/`)
2. Implement in `pkg/grpc/actions`
3. Regenerate protobuf and OpenAPI specs (see [API & Protobuf Changes](#api--protobuf-changes))
4. Add authorization in `pkg/authorization/interceptor.go`

### Adding Database Models

1. Create model in `pkg/models`
2. Create migration: `make db.migration.create NAME=add-model-name`
3. Implement both transaction and non-transaction variants of methods (see [Database Models](#database-models))

### Adding New Components or Triggers

See [Component Implementation Guide](../development/component-implementations.md) for detailed guidelines.

## Related Guides

- [Development Workflow](development-workflow.md) - Day-to-day workflow
- [Component Implementation Guide](../development/component-implementations.md) - Component patterns
- [E2E Testing Guide](../development/e2e_tests.md) - Testing guidelines

