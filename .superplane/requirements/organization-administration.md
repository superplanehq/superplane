# Organization Administration

## Overview

Organization administration lets authorized people manage organization
identity, membership, invitations, groups, roles, usage, and installation-level
settings. These related surfaces define who collaborates in SuperPlane and
what each collaborator may do.

## Terminology

- **Organization:** A tenant that contains members, Apps, integrations, and
  access policy.
- **Group:** A collection of members that can receive role-based access.
- **Invitation link:** A controlled route for joining an organization.

## Requirements

### REQ-ORG-001: Organization lifecycle and settings

**User story:** As an organization owner, I want to create and maintain an
organization, so that my team has an isolated SuperPlane workspace.

**Acceptance criteria:**

- **AC-ORG-001.1:** When an eligible user creates an organization with valid
  details, SuperPlane shall make it available in that user's organization
  selection.
- **AC-ORG-001.2:** When an authorized owner changes organization settings,
  SuperPlane shall display the saved values on the next visit.

### REQ-ORG-002: Membership and invitations

**User story:** As an organization administrator, I want to invite and remove
members, so that organization access follows team membership.

**Acceptance criteria:**

- **AC-ORG-002.1:** When an administrator enables or resets an invitation link,
  SuperPlane shall expose the current link and invalidate a replaced link.
- **AC-ORG-002.2:** When an authorized administrator removes a member,
  SuperPlane shall stop listing that person as an active organization member.

### REQ-ORG-003: Groups and roles

**User story:** As an access administrator, I want to group members and assign
roles, so that permissions can be managed consistently.

**Acceptance criteria:**

- **AC-ORG-003.1:** When an administrator adds or removes a member from a
  group, SuperPlane shall reflect the resulting membership in the group view.
- **AC-ORG-003.2:** When an administrator assigns a role, the affected
  principal shall gain allowed actions and remain blocked from actions outside
  that role.

### REQ-ORG-004: Administrative visibility and guards

**User story:** As an installation administrator, I want visibility into
organizations, accounts, settings, usage, and operational tasks, so that I can
govern the deployment.

**Acceptance criteria:**

- **AC-ORG-004.1:** When an installation administrator opens administration,
  SuperPlane shall present the administration areas that their privileges
  allow.
- **AC-ORG-004.2:** When a non-administrator requests an administrative
  mutation, SuperPlane shall reject it without changing organization state.

## Traceability

- **API evidence:** [organization service](../../protos/organizations.proto),
  [groups service](../../protos/groups.proto),
  [roles service](../../protos/roles.proto), and
  [usage service](../../protos/usage.proto)
- **UI evidence:** [organization and admin routes](../../web_src/src/App.tsx)
- **Behavior evidence:** [organization creation](../../test/e2e/organization_test.go),
  [invitations](../../test/e2e/invitations_test.go),
  [members](../../test/e2e/members_test.go),
  [groups](../../test/e2e/groups_test.go), and
  [roles](../../test/e2e/roles_test.go)
- **Feature blueprint:** [Identity, Organizations, and Access](../blueprints/features/identity-organizations-and-access.feature.md)

## Open Questions

- Which organization settings require owner-only access rather than
  administrator access?
- What usage limits and warnings must be visible before an action is blocked?
