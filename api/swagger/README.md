# Swagger/OpenAPI Specification

This directory contains the OpenAPI specifications for the Superplane API.

## Files

| File | Status | Description |
|------|--------|-------------|
| `superplane.swagger.json` | 🤖 **Generated** | Complete merged OpenAPI spec (DO NOT edit manually) |
| `account-auth.swagger.json` | 📝 **Source-controlled** | Manual definitions for auth/account/setup endpoints |
| `swagger-ui.html` | 📝 **Source-controlled** | Swagger UI for viewing the API documentation |
| `README.md` | 📝 **Source-controlled** | This file |

## How It Works

The OpenAPI specification is generated using a **hybrid approach** that combines auto-generated and manually-defined endpoints.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    make openapi.spec.gen                     │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────────┐
        │  scripts/protoc_openapi_spec.sh       │
        └───────────┬───────────────────────────┘
                    │
        ┌───────────┴──────────────┐
        │                          │
        ▼                          ▼
┌──────────────────┐      ┌──────────────────┐
│ Step 1: Generate │      │ Step 2: Merge    │
│ from protobuf    │      │ with manual      │
│                  │      │                  │
│ protos/*.proto   │      │ account-auth     │
│       ↓          │      │  .swagger.json   │
│ protoc           │      │       ↓          │
│       ↓          │      │ merge-swagger.js │
│ superplane       │  →   │  (Node.js)       │
│ .swagger.json    │      │       ↓          │
│ (temp)           │      │ superplane       │
└──────────────────┘      │ .swagger.json    │
                          │ (final merged)   │
                          └──────────────────┘
```

### 1. Auto-Generated Endpoints (from protobuf)

Most API endpoints are automatically generated from protobuf definitions using gRPC-Gateway:

- `/api/v1/organizations/{id}/*` - Organization-scoped operations
- `/api/v1/users/*` - User management
- `/api/v1/integrations/*` - Integration management
- `/api/v1/secrets/*` - Secret management
- `/api/v1/groups/*` - Group management
- `/api/v1/roles/*` - Role management
- `/api/v1/components/*` - Component management
- `/api/v1/triggers/*` - Trigger management
- `/api/v1/widgets/*` - Widget management
- `/api/v1/blueprints/*` - Blueprint management
- `/api/v1/workflows/*` - Workflow management
- And more...

**Source:** `protos/*.proto` files with gRPC service definitions and HTTP annotations.

**Command:**
```bash
make openapi.spec.gen
# or directly:
docker compose -f docker-compose.dev.yml run --rm --no-deps app \
  /app/scripts/protoc_openapi_spec.sh authorization,organizations,...
```

### 2. Manual Endpoints (non-protobuf)

Some endpoints don't fit the gRPC model and are defined manually in `account-auth.swagger.json`:

| Endpoint | Method | Purpose | Why Manual? |
|----------|--------|---------|-------------|
| `/auth/config` | GET | Get auth configuration | Simple config endpoint |
| `/login` | POST | Password login | Form-based auth, not gRPC-friendly |
| `/signup` | POST | User signup | Form-based auth, not gRPC-friendly |
| `/account` | GET | Current user info | Account-level (not org-scoped) |
| `/organizations` | GET/POST | List/create orgs | Account-level (not org-scoped) |
| `/api/v1/setup-owner` | POST | Initial setup | One-time setup flow |

**Why these are manual:**
- **OAuth flows** are inherently HTTP-based (redirects, callbacks, cookies)
- **Form-based login** uses `application/x-www-form-urlencoded`, not protobuf
- **Account-level operations** use different auth (account token) vs org-scoped (org context)
- **Setup flows** are special one-time initialization processes

### 3. Merging Process

The script `scripts/protoc_openapi_spec.sh` automatically merges both sources:

**Step 1: Generate from protobuf**
```bash
protoc --openapiv2_out=api/swagger \
       --openapiv2_opt=merge_file_name=superplane.swagger \
       protos/*.proto
# Creates: api/swagger/superplane.swagger.json (temporary)
```

**Step 2: Merge with manual swagger**
```bash
node scripts/merge-swagger.js \
  api/swagger/superplane.swagger.json \       # Generated (temp)
  api/swagger/account-auth.swagger.json \     # Manual
  api/swagger/temp-merged.json                # Output

mv temp-merged.json superplane.swagger.json   # Replace
# Result: api/swagger/superplane.swagger.json (final merged)
```

**What gets merged:**
- ✅ `paths` - All endpoint definitions
- ✅ `definitions` - All schema definitions
- ✅ `tags` - All tags (duplicates removed)
- ✅ `securityDefinitions` - Authentication schemes

## Making Changes

### For protobuf-based endpoints (most endpoints):

1. **Edit the appropriate `.proto` file** in `protos/`
   ```bash
   # Example: Add new endpoint to organizations
   vim protos/organizations.proto
   ```

2. **Regenerate everything**
   ```bash
   make gen
   # This runs: pb.gen, openapi.spec.gen, openapi.client.gen, format
   ```

3. **Verify the changes**
   ```bash
   # Check the endpoint is present
   jq '.paths | keys' api/swagger/superplane.swagger.json | grep "your-new-endpoint"

   # Check the web client was updated
   ls -la web_src/src/api-client/
   ```

### For manual auth/account endpoints (6 endpoints):

1. **Edit `api/swagger/account-auth.swagger.json`**
   ```bash
   vim api/swagger/account-auth.swagger.json
   ```

2. **Regenerate and merge**
   ```bash
   make openapi.spec.gen
   ```

3. **Verify the merged output**
   ```bash
   jq '.paths["/your-endpoint"]' api/swagger/superplane.swagger.json
   ```

**Example: Adding a new auth endpoint**
```json
{
  "paths": {
    "/auth/reset-password": {
      "post": {
        "summary": "Request password reset",
        "operationId": "Auth_ResetPassword",
        "parameters": [
          {
            "name": "email",
            "in": "formData",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "200": {"description": "Reset email sent"}
        },
        "tags": ["Authentication"]
      }
    }
  }
}
```

## Script Configuration

The generation script uses these variables (defined at the top of `scripts/protoc_openapi_spec.sh`):

```bash
# Configuration
OPENAPI_OUT=api/swagger                       # Output directory
PROTO_DIR="protos"                            # Proto source directory

# Output filenames (without .json extension)
MERGE_FILE_NAME=superplane.swagger            # Auto-generated swagger base name
MANUAL_SWAGGER_FILE=account-auth.swagger      # Manual swagger base name
```

**Why `.json` is added later:**
- Protoc creates `superplane.swagger.json` automatically
- We add `.json` when constructing full paths for consistency
- Both variables follow the same pattern (no extension at definition)

To change filenames, just update these variables at the top of the script.

## Testing

After making changes, verify everything works:

### 1. Basic merge test
```bash
make openapi.spec.gen
# Should output: "Successfully merged swagger files"
```

### 2. Verify endpoints
```bash
# Check total endpoint count
jq '.paths | length' api/swagger/superplane.swagger.json

# List all manual endpoints
jq -r '.paths | keys[]' api/swagger/superplane.swagger.json | \
  grep -E '^/(auth|login|signup|account|organizations|api/v1/setup)'

# Check specific endpoint
jq '.paths["/api/v1/setup-owner"]' api/swagger/superplane.swagger.json
```

### 3. Verify definitions
```bash
# Check total definition count
jq '.definitions | length' api/swagger/superplane.swagger.json

# List manual definitions
jq '.definitions | keys[]' api/swagger/superplane.swagger.json | \
  grep -E '^(Account|Auth|SetupOwner)'
```

### 4. Generate web client
```bash
cd web_src
npm run generate:api
# Should generate: src/api-client/types.gen.ts and src/api-client/sdk.gen.ts
```

### 5. Verify TypeScript types
```bash
# Check that manual endpoints have types
grep -E '(Account|AuthConfig|SetupOwner)' web_src/src/api-client/types.gen.ts
```

## Troubleshooting

### Problem: "Manual swagger file not found"
```
Manual swagger file not found at api/swagger/account-auth.swagger.json, skipping merge
```
**Solution:** The manual swagger file is missing. Create it or check the `MANUAL_SWAGGER_FILE` variable.

### Problem: Merge fails with "Cannot read property"
```
Error merging swagger files: Cannot read property 'paths' of undefined
```
**Solution:** One of the swagger files has invalid JSON. Validate with:
```bash
jq . api/swagger/account-auth.swagger.json
jq . api/swagger/superplane.swagger.json
```

### Problem: Manual endpoints missing after merge
**Solution:** Check that `merge_manual_swagger` function is being called:
```bash
# Should see "Merging manual swagger with generated swagger" in output
make openapi.spec.gen 2>&1 | grep -i merge
```

### Problem: Duplicate tags in output
**Solution:** The merge script already handles this. If you see duplicates, check that you're using the latest version of `scripts/merge-swagger.js`.

## Dependencies

The build process requires:

| Dependency | Purpose | Installation |
|------------|---------|--------------|
| `node` | Run merge script | ✅ Already installed (for `web_src`) |
| `protoc` | Compile `.proto` files | ✅ Installed in Docker base image |
| `protoc-gen-go` | Generate Go code from proto | ✅ Installed in Docker dev image |
| `protoc-gen-go-grpc` | Generate Go gRPC code | ✅ Installed in Docker dev image |
| `protoc-gen-grpc-gateway` | Generate gRPC-Gateway code | ✅ Installed in Docker dev image |
| `protoc-gen-openapiv2` | Generate OpenAPI spec | ✅ Installed in Docker dev image |

**No additional dependencies needed!** Everything runs in the Docker container.

## Why This Approach?

We chose **Option 1** (separate swagger + merge) over migrating everything to gRPC because:

### ✅ Pros
1. **OAuth flows** are inherently HTTP-based (redirects, cookies, OAuth dance)
2. **Form-based login** is standard HTML form submission, not protobuf
3. **Quick to implement** - no need to learn gRPC for simple endpoints
4. **Separation of concerns** - Auth logic stays in simple HTTP handlers
5. **No breaking changes** - Existing services continue to work
6. **Easy to maintain** - JavaScript code anyone can understand

### ❌ Why not full gRPC for auth?
- OAuth callbacks can't be protobuf (they're HTTP redirects)
- Cookie-based sessions don't fit the gRPC model
- Form data (login/signup) is `application/x-www-form-urlencoded`
- Setup flows are one-time operations, not worth gRPC ceremony

### The Best of Both Worlds
- **Resource APIs** (orgs, users, etc.) → gRPC-Gateway (type-safe, auto-generated)
- **Auth & setup** → Direct HTTP (simple, pragmatic)
- **One merged swagger** → Consistent API documentation

## Related Files

- `scripts/protoc_openapi_spec.sh` - Main generation script
- `scripts/merge-swagger.js` - Node.js merge script
- `Makefile` - Build commands (`make openapi.spec.gen`, `make gen`)
- `web_src/openapi-ts.config.ts` - TypeScript client generator config
- `web_src/src/api-client/` - Generated TypeScript client (auto-generated)
- `web_src/src/services/authService.ts` - Manual auth service (uses manual endpoints)
- `web_src/src/services/organizationService.ts` - Manual org service (uses manual endpoints)

## Further Reading

- [gRPC-Gateway Documentation](https://grpc-ecosystem.github.io/grpc-gateway/)
- [OpenAPI Specification v2](https://swagger.io/specification/v2/)
- [Protobuf HTTP Annotations](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto)

