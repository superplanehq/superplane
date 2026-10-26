# Shared invite links

SuperPlane no longer creates shared invite links on any plan or installation.
Administrators can no longer view, enable, disable, or reset shared links.
Shared links no longer admit new organization members or create accounts.

## Removed interfaces

This release removes the shared invite link management interfaces. Remove
calls to them from scripts and API clients before you upgrade.

| Interface | Removed item |
| --- | --- |
| gRPC | `Organizations.GetInviteLink` |
| gRPC | `Organizations.UpdateInviteLink` |
| gRPC | `Organizations.ResetInviteLink` |
| REST | `GET /api/v1/organizations/{id}/invite-link` |
| REST | `PATCH /api/v1/organizations/{id}/invite-link` |
| REST | `POST /api/v1/organizations/{id}/invite-link/reset` |
| CLI | `superplane members invite-link get` |
| CLI | `superplane members invite-link update` |
| CLI | `superplane members invite-link reset` |

The generated API clients no longer include these methods.

## Old shared links

SuperPlane keeps the stored shared links. The upgrade does not delete them.
`POST /api/v1/invite-links/{token}/accept` is still available.

- Existing members can still open their organization through enabled old
  shared links.
- Disabled old links stay disabled. They also stay disabled for existing
  members.
- Old links do not admit new members, enabled or disabled.

## Email invitations

Use email invitations to add people to an organization.
You can copy each individual invitation link and send it to the invited person.
This replaces delivery of the invitation email only. New invited users who create
password accounts still need email delivery to receive a verification code.

Each invitation applies to the invited email address.
Invited people can still create an account with that email address when public
signups are closed.
