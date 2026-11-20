# SuperPlane Enterprise code

Code in this directory is subject to the SuperPlane Enterprise Edition License
in [`LICENSE`](LICENSE). All other code is licensed under the Apache License,
Version 2.0.

SuperPlane is one binary. Enterprise code is always in the binary. Licensed
features run only when the installation license grants the feature. Billing
code in `ee/billing` is covered by the Enterprise source license. Billing does
not require an installation license or a runtime license feature. The server
always installs the billing implementation, including when billing is disabled,
so usage costs remain recorded. For installation license sources,
verification, and the admin endpoints, see
[`docs/contributing/enterprise-licensing.md`](../docs/contributing/enterprise-licensing.md).

## Rules

- Core code must not import `ee/`. Only `pkg/server` registers an Enterprise
  implementation.
- Billing uses the interfaces in `pkg/billing` and the billing capability in
  `pkg/enterprise`. It does not use the runtime license checks below.
- `licensing.Service` answers whether an organization can use a feature.
  `enterprise.Registry` answers which code runs. Keep those separate.
- Entitlement checks take the organization. `licensing.Allows` and
  `licensing.Require` receive the context and the organization ID. A feature
  is available only when the license is active, the license contains the
  feature, and the organization has a grant. A nil organization ID is not
  entitled.
- For licensed capabilities, core registers a Community implementation that
  refuses licensed operations. `pkg/server` replaces it with the EE implementation.
  The billing capability has no fallback. A missing implementation is an error.
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
   func (s *Service) CreateThing(ctx context.Context, orgID uuid.UUID, req *pb.CreateThingRequest) (*pb.CreateThingResponse, error) {
       if err := licensing.Require(ctx, s.entitlements, orgID, licensing.FeatureThing); err != nil {
           return nil, grpcerrors.PermissionDenied(err, err.Error())
       }

       return createThing(ctx, req)
   }
   ```

   Resolve `orgID` from the organization in the request. If the organization
   cannot be resolved, deny the request. A missing grant uses the same error
   as a missing license.

4. **Register the implementation.** In `pkg/server`, register the `ee/`
   service on the enterprise registry under the key from step 2. This is the
   only place that imports the new package.

   ```go
   features.Register(enterprise.Things, thing.NewService(licenseService))
   ```

5. **Tag the gateway rules.** Add the feature to `RequiredLicenseFeatures` on
   each rule in `pkg/authorization/gateway_auth_rules.go` that creates or
   expands access. The rule domain type must be `models.DomainTypeOrganization`.
   The gateway checks the grant for that organization:

   ```go
   {Method: "POST", Pattern: "/api/v1/things"}: {
       Resource:                "things",
       Action:                  "create",
       DomainType:              models.DomainTypeOrganization,
       RequiredLicenseFeatures: []licensing.Feature{licensing.FeatureThing},
   },
   ```

6. **Write the tests.** Test each operation with and without the grant.
   Make sure that read, delete, and revoke operations work without a grant.
   A grant in another organization does not allow the operation.

   - Unit tests: use `licensingtest.EnterpriseService(licensing.FeatureThing)`
     when every organization should be allowed. Use
     `licensingtest.EnterpriseServiceForOrganizations` when the test needs
     one organization to have the grant and another organization to lack it.
     Use `licensing.Community` for Community mode.
   - Gateway rules: add the routes to
     `pkg/authorization/license_features_test.go`. Keep the check that every
     rule with `RequiredLicenseFeatures` uses the organization domain.
   - E2E tests: grant the feature to the session organization. A new
     organization does not receive grants.

7. **Update the UI.** Add the feature to `ENTERPRISE_FEATURES` in
   `web_src/src/lib/license.ts`, so that the License page shows it. Inside an
   organization, call `hasOrganizationEnterpriseFeature` with the
   organization's `enterpriseFeatures`. Do not read
   `account.license.features` for that check. The License page and the
   expiry banner keep `account.license`. The UI check is only for display.
   The backend checks enforce the license.

8. **Update the documentation.** Add the feature and its operations to the
   feature table in `docs/contributing/enterprise-licensing.md`.

## Development

`make dev.server` uses a development license when no license is installed.
That license allows every recognized Enterprise feature in every
organization and does not write grants. A new feature works after step 1.
A real license installed in development uses the normal grant rules. To test
Community mode, set `SUPERPLANE_LICENSE_DEV_ENTERPRISE=false` in `.env` and
restart the server.

Tests sign licenses with keys that exist only in memory. Do not commit private
keys or license files.
