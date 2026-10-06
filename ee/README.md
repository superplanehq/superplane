# SuperPlane Enterprise code

Code in this directory is subject to the SuperPlane Enterprise Edition License
in [`LICENSE`](LICENSE). All other code is licensed under the Apache License,
Version 2.0.

SuperPlane is one binary. Enterprise code is always in the binary, but it runs
only when the installation license grants its feature. For license sources,
verification, and the admin endpoints, see
[`docs/contributing/enterprise-licensing.md`](../docs/contributing/enterprise-licensing.md).

## Rules

- Core code must not import `ee/`. Only `pkg/server` registers an Enterprise
  implementation.
- `licensing.Service` answers whether a license grants a feature.
  `enterprise.Registry` answers which code runs. Keep those separate.
- The core registers a Community implementation of each capability. It refuses
  every Enterprise operation. `pkg/server` replaces that registration with the
  `ee/` implementation.
- Check the license two times. Each `ee/` method checks the license, and the
  HTTP gateway rule checks `RequiredLicenseFeatures`. Do not remove one check
  because the other check exists.
- Require a license only for operations that create or expand Enterprise
  access. Do not require a license to read, delete, or revoke. Administrators
  must be able to remove access after a license expires.
- Return `PermissionDenied` with `licensing.ErrNotLicensed`. Do not show
  license details to users who are not installation administrators.

## Add a new Enterprise feature

1. **Define the feature key.** Add a `Feature` constant and add it to
   `recognizedFeatures` in `pkg/licensing/features.go`. A key is permanent.
   Do not rename or reuse a key. Ask a licensing administrator to create a
   feature with the same key in the license issuer.

2. **Define the core interface and register it.** Put the interface in
   `pkg/enterprise`. Add a Community implementation that refuses the
   operation, and register it from `NewRegistry` under a new key. Callers
   resolve it with `enterprise.Get`. `Rbac` is the example: `custom_roles`
   and `groups` share that one implementation. A new feature gets its own
   interface and key. Do not add its methods to `Rbac`.

   ```go
   impl, err := enterprise.Get[Thing](registry, enterprise.Things)
   ```

3. **Implement the feature in `ee/`.** Add a package such as
   `ee/<feature>/`. Each public method checks the license before it does
   work:

   ```go
   func (s *Service) CreateThing(ctx context.Context, req *pb.CreateThingRequest) (*pb.CreateThingResponse, error) {
       if err := licensing.Require(s.entitlements, licensing.FeatureThing); err != nil {
           return nil, grpcerrors.PermissionDenied(err, err.Error())
       }

       return createThing(ctx, req)
   }
   ```

4. **Register the implementation.** In `pkg/server`, register the `ee/`
   service on the enterprise registry under the key from step 2. This is the
   only place that imports the new package.

   ```go
   features.Register(enterprise.Things, thing.NewService(licenseService))
   ```

5. **Tag the gateway rules.** Add the feature to `RequiredLicenseFeatures` on
   each rule in `pkg/authorization/gateway_auth_rules.go` that creates or
   expands access:

   ```go
   {Method: "POST", Pattern: "/api/v1/things"}: {
       Resource:                "things",
       Action:                  "create",
       DomainType:              models.DomainTypeOrganization,
       RequiredLicenseFeatures: []licensing.Feature{licensing.FeatureThing},
   },
   ```

6. **Write the tests.** Test each operation with and without the feature.
   Make sure that read, delete, and revoke operations work without a license.

   - Unit tests: use `licensingtest.EnterpriseService(licensing.FeatureThing)`
     for a licensed installation and `licensing.Community` for Community mode.
   - Gateway rules: add the routes to
     `pkg/authorization/license_features_test.go`.
   - E2E tests: add the feature to `server.SetLicenseServiceForTests` in
     `test/e2e/test_context.go`.

7. **Update the documentation.** Add the feature and its operations to the
   feature table in `docs/contributing/enterprise-licensing.md`.

## Development

The development server grants every recognized Enterprise feature, so a new
feature works in `make dev.server` after step 1. To test Community mode, set
`SUPERPLANE_LICENSE_DEV_ENTERPRISE=false` in `.env` and restart the server.

Tests sign licenses with keys that exist only in memory. Do not commit private
keys or license files.
