# Identity and Access

## Overview

SuperPlane must give human users and programmatic clients a secure way to
authenticate, select an organization, and act only within granted permissions.
This feature groups sign-in, owner setup, personal access tokens, API keys, and
permission enforcement because they form one access boundary.

## Terminology

- **Owner setup:** First-run creation or promotion of the installation owner.
- **API key:** Organization-scoped credential for programmatic access.
- **Role:** A named collection of allowed actions.

## Requirements

### REQ-IAM-001: Human authentication and organization entry

**User story:** As an organization member, I want to authenticate and enter an
organization I belong to, so that I can reach authorized SuperPlane features.

**Acceptance criteria:**

- **AC-IAM-001.1:** When a valid member completes a supported sign-in flow,
  SuperPlane shall show the organization selector or the selected
  organization's authorized landing page.
- **AC-IAM-001.2:** When an unauthenticated visitor requests a protected App or
  settings page, SuperPlane shall require authentication before exposing its
  content.

### REQ-IAM-002: Installation owner bootstrap

**User story:** As an installation administrator, I want to establish the first
owner, so that a new SuperPlane installation can be governed.

**Acceptance criteria:**

- **AC-IAM-002.1:** When owner setup is required, SuperPlane shall direct the
  administrator to setup before allowing normal protected navigation.
- **AC-IAM-002.2:** When owner setup succeeds, SuperPlane shall recognize the
  resulting owner as authorized to administer the installation.

### REQ-IAM-003: Programmatic credentials

**User story:** As an organization administrator, I want to manage API keys, so
that automation can access SuperPlane without sharing a browser session.

**Acceptance criteria:**

- **AC-IAM-003.1:** When an administrator creates or regenerates an API key,
  SuperPlane shall reveal the usable token at that time and identify the key in
  subsequent listings without exposing the token.
- **AC-IAM-003.2:** When an administrator deletes an API key, subsequent
  requests using that key shall no longer be authenticated.

### REQ-IAM-004: Consistent authorization

**User story:** As an organization viewer, I want restricted actions to remain
unavailable across navigation and direct requests, so that my access matches
my assigned permissions.

**Acceptance criteria:**

- **AC-IAM-004.1:** When a viewer lacks permission for a protected action,
  SuperPlane shall not complete that action even if its address is requested
  directly.
- **AC-IAM-004.2:** When a user's permissions permit viewing but not editing,
  SuperPlane shall preserve readable content while preventing mutations.

### REQ-IAM-005: Non-human service identities

**User story:** As an organization administrator, I want automation to use a
named non-human identity with assigned access, so that credentials are not tied
to an individual member.

**Acceptance criteria:**

- **AC-IAM-005.1:** When an administrator creates a service identity,
  SuperPlane shall issue its credential once and allow an eligible role to be
  assigned without granting organization ownership.
- **AC-IAM-005.2:** When the service identity is deleted or its credential is
  regenerated, the prior credential shall stop authenticating requests.

## Traceability

- **Product context:** [README security and agent access](../../README.md#security)
- **Product direction:** [service-account requirements](../../docs/prd/service-accounts.md)
- **API evidence:** [API keys service](../../protos/api_keys.proto),
  [current-user service](../../protos/me.proto), and
  [authorization contract](../../protos/authorization.proto)
- **UI evidence:** [protected routes and setup guard](../../web_src/src/App.tsx)
- **Behavior evidence:** [owner setup](../../test/e2e/owner_setup_test.go),
  [login](../../test/e2e/login_page_test.go), and
  [API key management](../../test/e2e/api_keys_test.go)
- **Feature blueprint:** [Identity, Organizations, and Access](../blueprints/features/identity-organizations-and-access.feature.md)

## Open Questions

- Which additional sign-in methods are part of the supported product contract?
- Will programmatic identities have token expiration and overlapping rotation?
