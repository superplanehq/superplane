import { useEffect, useState } from "react";
import type { UseMutationResult } from "@tanstack/react-query";
import type { OrganizationsOrganization } from "@/api-client/types.gen";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  oauthDraftFromAllowedProviders,
  oauthProvidersListEqual,
  oauthProvidersToSave,
  oauthSavedPolicySummary,
} from "../invitationOauth";
import { OAuthInvitationSettingsPanel } from "./OAuthInvitationSettingsPanel";

type UpdateOrganizationMutation = UseMutationResult<unknown, Error, { allowedOauthProviders?: string[] }, unknown>;

export interface OAuthInvitationSettingsCardProps {
  organization: OrganizationsOrganization;
  organizationId: string;
  canUpdateOrg: boolean;
  permissionsLoading: boolean;
  updateOrganizationMutation: UpdateOrganizationMutation;
}

export function OAuthInvitationSettingsCard({
  organization,
  organizationId,
  canUpdateOrg,
  permissionsLoading,
  updateOrganizationMutation,
}: OAuthInvitationSettingsCardProps) {
  const [oauthRestrictProviders, setOauthRestrictProviders] = useState(() => {
    const d = oauthDraftFromAllowedProviders(organization.spec?.allowedOauthProviders?.providers);
    return d.restrict;
  });
  const [oauthGithub, setOauthGithub] = useState(() => {
    const d = oauthDraftFromAllowedProviders(organization.spec?.allowedOauthProviders?.providers);
    return d.github;
  });
  const [oauthGoogle, setOauthGoogle] = useState(() => {
    const d = oauthDraftFromAllowedProviders(organization.spec?.allowedOauthProviders?.providers);
    return d.google;
  });
  const [oauthMessage, setOauthMessage] = useState<string | null>(null);
  const [oauthSelectionError, setOauthSelectionError] = useState<string | null>(null);

  useEffect(() => {
    const d = oauthDraftFromAllowedProviders(organization.spec?.allowedOauthProviders?.providers);
    setOauthRestrictProviders(d.restrict);
    setOauthGithub(d.github);
    setOauthGoogle(d.google);
    setOauthSelectionError(null);
  }, [organization.spec?.allowedOauthProviders?.providers]);

  const serverOauthList = organization.spec?.allowedOauthProviders?.providers ?? [];
  const draftOauthList = oauthProvidersToSave(oauthRestrictProviders, oauthGithub, oauthGoogle);
  const oauthHasUnsavedChanges = !oauthProvidersListEqual(serverOauthList, draftOauthList);
  const policySummary = oauthSavedPolicySummary(organization.spec?.allowedOauthProviders?.providers);

  const handleSaveOAuthProviders = () => {
    void (async () => {
      if (!canUpdateOrg || !organizationId) {
        return;
      }
      if (oauthRestrictProviders && !oauthGithub && !oauthGoogle) {
        setOauthSelectionError("Turn off the switch to allow any provider, or select at least one provider.");
        return;
      }
      setOauthSelectionError(null);
      setOauthMessage(null);
      try {
        await updateOrganizationMutation.mutateAsync({ allowedOauthProviders: draftOauthList });
        setOauthMessage("OAuth invitation settings saved");
        setTimeout(() => setOauthMessage(null), 3000);
      } catch {
        setOauthMessage("Failed to save OAuth invitation settings");
        setTimeout(() => setOauthMessage(null), 3000);
      }
    })();
  };

  return (
    <PermissionTooltip
      allowed={canUpdateOrg || permissionsLoading}
      message="You don't have permission to update this organization."
      className="w-full"
    >
      <OAuthInvitationSettingsPanel
        policySummary={policySummary}
        oauthHasUnsavedChanges={oauthHasUnsavedChanges}
        oauthRestrictProviders={oauthRestrictProviders}
        setOauthRestrictProviders={setOauthRestrictProviders}
        oauthGithub={oauthGithub}
        setOauthGithub={setOauthGithub}
        oauthGoogle={oauthGoogle}
        setOauthGoogle={setOauthGoogle}
        setOauthSelectionError={setOauthSelectionError}
        oauthSelectionError={oauthSelectionError}
        oauthMessage={oauthMessage}
        canUpdateOrg={canUpdateOrg}
        isPending={updateOrganizationMutation.isPending}
        onSave={handleSaveOAuthProviders}
      />
    </PermissionTooltip>
  );
}
