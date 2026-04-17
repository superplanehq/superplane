package members

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func organizationDomainType() openapi_client.AuthorizationDomainType {
	return openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION
}

// splitUserIdentifier picks between a user id and a user email. A positional
// containing "@" is treated as an email, so CLI commands can accept either
// form in the positional slot. Returns (userID, userEmail, error). Exactly one
// non-empty on success; both empty when no identifier was provided; error
// when the caller supplied both a positional and --email.
func splitUserIdentifier(positional string, emailFlag string) (string, string, error) {
	positional = strings.TrimSpace(positional)
	emailFlag = strings.TrimSpace(emailFlag)

	if positional != "" && emailFlag != "" {
		return "", "", fmt.Errorf("pass either a positional user id or --email, not both")
	}

	if positional != "" {
		if strings.Contains(positional, "@") {
			return "", positional, nil
		}
		return positional, "", nil
	}
	if emailFlag != "" {
		return "", emailFlag, nil
	}
	return "", "", nil
}

// findMember locates a user by id or email within the organization's member list.
// Returns the user and true if found. The search is case-insensitive for emails.
func findMember(ctx core.CommandContext, organizationID string, identifier string) (openapi_client.SuperplaneUsersUser, bool, error) {
	response, _, err := ctx.API.UsersAPI.
		UsersListUsers(ctx.Context).
		DomainType(string(organizationDomainType())).
		DomainId(organizationID).
		IncludeRoles(true).
		Execute()
	if err != nil {
		return openapi_client.SuperplaneUsersUser{}, false, err
	}

	trimmed := strings.TrimSpace(identifier)
	needle := strings.ToLower(trimmed)
	for _, user := range response.GetUsers() {
		metadata := user.GetMetadata()
		if metadata.GetId() == trimmed {
			return user, true, nil
		}
		if strings.EqualFold(metadata.GetEmail(), needle) {
			return user, true, nil
		}
	}

	return openapi_client.SuperplaneUsersUser{}, false, nil
}

// resolveMember returns the organization member matching the positional identifier
// (id) and/or --email flag. It errors if neither is provided, both are provided,
// or the member cannot be found.
func resolveMember(ctx core.CommandContext, organizationID string, positional string, emailFlag string) (openapi_client.SuperplaneUsersUser, error) {
	positional = strings.TrimSpace(positional)
	emailFlag = strings.TrimSpace(emailFlag)

	if positional != "" && emailFlag != "" {
		return openapi_client.SuperplaneUsersUser{}, fmt.Errorf("pass either a positional user id or --email, not both")
	}

	identifier := positional
	if identifier == "" {
		identifier = emailFlag
	}
	if identifier == "" {
		return openapi_client.SuperplaneUsersUser{}, fmt.Errorf("a user id or --email is required")
	}

	user, found, err := findMember(ctx, organizationID, identifier)
	if err != nil {
		return openapi_client.SuperplaneUsersUser{}, err
	}
	if !found {
		return openapi_client.SuperplaneUsersUser{}, fmt.Errorf("member %q not found in organization", identifier)
	}
	return user, nil
}

func roleNames(user openapi_client.SuperplaneUsersUser) []string {
	status := user.GetStatus()
	names := make([]string, 0, len(status.GetRoles()))
	for _, role := range status.GetRoles() {
		if role.HasRoleName() {
			names = append(names, role.GetRoleName())
		}
	}
	return names
}

func renderMemberListText(stdout io.Writer, users []openapi_client.SuperplaneUsersUser) error {
	if len(users) == 0 {
		_, err := fmt.Fprintln(stdout, "No members found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tEMAIL\tNAME\tROLES\tCREATED_AT")

	for _, user := range users {
		metadata := user.GetMetadata()
		spec := user.GetSpec()

		createdAt := ""
		if metadata.HasCreatedAt() {
			createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
		}

		roles := strings.Join(roleNames(user), ",")
		if roles == "" {
			roles = "-"
		}

		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\n",
			metadata.GetId(),
			metadata.GetEmail(),
			spec.GetDisplayName(),
			roles,
			createdAt,
		)
	}

	return writer.Flush()
}

func renderMemberText(stdout io.Writer, user openapi_client.SuperplaneUsersUser) error {
	metadata := user.GetMetadata()
	spec := user.GetSpec()
	status := user.GetStatus()

	_, _ = fmt.Fprintf(stdout, "ID: %s\n", metadata.GetId())
	_, _ = fmt.Fprintf(stdout, "Email: %s\n", metadata.GetEmail())
	_, _ = fmt.Fprintf(stdout, "Name: %s\n", spec.GetDisplayName())
	if metadata.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", metadata.GetCreatedAt().Format(time.RFC3339))
	}

	_, _ = fmt.Fprintln(stdout, "Roles:")
	if len(status.GetRoles()) == 0 {
		_, _ = fmt.Fprintln(stdout, "  (none)")
	}
	for _, role := range status.GetRoles() {
		name := role.GetRoleName()
		display := role.GetRoleDisplayName()
		if display != "" && display != name {
			_, _ = fmt.Fprintf(stdout, "- %s (%s)\n", name, display)
		} else {
			_, _ = fmt.Fprintf(stdout, "- %s\n", name)
		}
	}

	providers := status.GetAccountProviders()
	if len(providers) > 0 {
		_, _ = fmt.Fprintln(stdout, "Account Providers:")
		for _, p := range providers {
			_, _ = fmt.Fprintf(stdout, "- %s\n", p.GetProviderType())
		}
	}

	return nil
}

func renderInvitationListText(stdout io.Writer, invitations []openapi_client.OrganizationsInvitation) error {
	if len(invitations) == 0 {
		_, err := fmt.Fprintln(stdout, "No invitations found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tEMAIL\tSTATE\tCREATED_AT")

	for _, invitation := range invitations {
		createdAt := ""
		if invitation.HasCreatedAt() {
			createdAt = invitation.GetCreatedAt().Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			invitation.GetId(),
			invitation.GetEmail(),
			invitation.GetState(),
			createdAt,
		)
	}

	return writer.Flush()
}

func renderInvitationText(stdout io.Writer, invitation openapi_client.OrganizationsInvitation) error {
	_, _ = fmt.Fprintf(stdout, "ID: %s\n", invitation.GetId())
	_, _ = fmt.Fprintf(stdout, "Email: %s\n", invitation.GetEmail())
	_, _ = fmt.Fprintf(stdout, "State: %s\n", invitation.GetState())
	_, _ = fmt.Fprintf(stdout, "Organization ID: %s\n", invitation.GetOrganizationId())
	if invitation.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", invitation.GetCreatedAt().Format(time.RFC3339))
	}
	return nil
}

func renderInviteLinkText(stdout io.Writer, link openapi_client.OrganizationsInviteLink) error {
	_, _ = fmt.Fprintf(stdout, "ID: %s\n", link.GetId())
	_, _ = fmt.Fprintf(stdout, "Enabled: %t\n", link.GetEnabled())
	_, _ = fmt.Fprintf(stdout, "Token: %s\n", link.GetToken())
	_, _ = fmt.Fprintf(stdout, "Organization ID: %s\n", link.GetOrganizationId())
	if link.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", link.GetCreatedAt().Format(time.RFC3339))
	}
	if link.HasUpdatedAt() {
		_, _ = fmt.Fprintf(stdout, "Updated At: %s\n", link.GetUpdatedAt().Format(time.RFC3339))
	}
	return nil
}
