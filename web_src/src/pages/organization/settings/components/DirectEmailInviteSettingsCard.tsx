import { useEffect, useState } from "react";
import type { UseMutationResult } from "@tanstack/react-query";
import type { OrganizationsOrganization } from "@/api-client/types.gen";
import { Fieldset, Label } from "@/components/Fieldset/fieldset";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Switch } from "@/ui/switch";

type UpdateOrganizationMutation = UseMutationResult<
  unknown,
  Error,
  { allowDirectEmailInviteCompletion?: boolean },
  unknown
>;

export interface DirectEmailInviteSettingsCardProps {
  organization: OrganizationsOrganization;
  organizationId: string;
  canUpdateOrg: boolean;
  permissionsLoading: boolean;
  updateOrganizationMutation: UpdateOrganizationMutation;
}

export function DirectEmailInviteSettingsCard({
  organization,
  organizationId,
  canUpdateOrg,
  permissionsLoading,
  updateOrganizationMutation,
}: DirectEmailInviteSettingsCardProps) {
  const [allowDirectEmailInviteCompletion, setAllowDirectEmailInviteCompletion] = useState(
    organization.spec?.allowDirectEmailInviteCompletion ?? true,
  );
  const [directEmailInviteMessage, setDirectEmailInviteMessage] = useState<string | null>(null);

  useEffect(() => {
    setAllowDirectEmailInviteCompletion(organization.spec?.allowDirectEmailInviteCompletion ?? true);
  }, [organization.spec?.allowDirectEmailInviteCompletion]);

  const handleAllowDirectEmailInviteToggle = async (enabled: boolean) => {
    if (!canUpdateOrg || !organizationId) {
      return;
    }

    const previous = allowDirectEmailInviteCompletion;
    setAllowDirectEmailInviteCompletion(enabled);
    setDirectEmailInviteMessage(null);

    try {
      await updateOrganizationMutation.mutateAsync({
        allowDirectEmailInviteCompletion: enabled,
      });
      setDirectEmailInviteMessage(
        enabled
          ? "Continue with email and password can complete pending invitations."
          : "Pending invitations require an allowed OAuth sign-in.",
      );
      setTimeout(() => setDirectEmailInviteMessage(null), 3000);
    } catch {
      setAllowDirectEmailInviteCompletion(previous);
      setDirectEmailInviteMessage("Failed to update invitation sign-in policy");
      setTimeout(() => setDirectEmailInviteMessage(null), 3000);
    }
  };

  return (
    <PermissionTooltip
      allowed={canUpdateOrg || permissionsLoading}
      message="You don't have permission to update this organization."
      className="w-full"
    >
      <Fieldset
        className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6"
        data-testid="non-oauth-invitation-settings-card"
      >
        <div className="flex items-start justify-between gap-6">
          <div>
            <Label
              htmlFor="organization-direct-email-invite-switch"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Continue with email and password
            </Label>
            <p className="text-sm text-gray-500 dark:text-gray-400 max-w-prose">
              Applies when someone has a pending email invitation and signs in without GitHub or Google: the{" "}
              <strong>Continue with email</strong> button on the login page (you get a sign-in link or a code by email),
              or email plus password when password login is enabled for your installation.
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-prose">
              When off, completing those invitations requires sign-in with an OAuth provider allowed in the section
              above.
            </p>
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {allowDirectEmailInviteCompletion ? "Allowed" : "OAuth only"}
            </span>
            <Switch
              id="organization-direct-email-invite-switch"
              checked={allowDirectEmailInviteCompletion}
              onCheckedChange={(checked: boolean) => {
                void handleAllowDirectEmailInviteToggle(checked);
              }}
              disabled={updateOrganizationMutation.isPending || !canUpdateOrg}
              aria-label="Toggle whether Continue with email or password sign-in can complete pending email invitations"
            />
          </div>
        </div>
        {directEmailInviteMessage ? (
          <p
            className={`mt-3 text-sm ${directEmailInviteMessage.includes("Failed") ? "text-red-600" : "text-green-600"}`}
          >
            {directEmailInviteMessage}
          </p>
        ) : null}
      </Fieldset>
    </PermissionTooltip>
  );
}
