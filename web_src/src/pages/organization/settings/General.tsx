import { useEffect, useState, type ChangeEvent } from "react";
import { Trash2 } from "lucide-react";
import { useParams } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import type { OrganizationsOrganization } from "../../../api-client/types.gen";
import { Field, Fieldset, Label } from "../../../components/Fieldset/fieldset";
import { Heading } from "../../../components/Heading/heading";
import { Input } from "../../../components/Input/input";
import { useDeleteOrganization, useUpdateOrganization } from "../../../hooks/useOrganizationData";
import { LoadingButton } from "@/components/ui/loading-button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Switch } from "@/ui/switch";
import { Checkbox } from "@/ui/checkbox";
import { usePermissions } from "@/contexts/PermissionsContext";
import { isChangeManagementSettingsEnabled } from "@/lib/env";

interface GeneralProps {
  organization: OrganizationsOrganization;
}

/** Local OAuth-invite controls derived from API `allowedOauthProviders` (must match useEffect sync logic). */
function oauthDraftFromAllowedProviders(providers: string[] | undefined) {
  const list = providers ?? [];
  if (list.length === 0) {
    return { restrict: false, github: true, google: true };
  }
  return { restrict: true, github: list.includes("github"), google: list.includes("google") };
}

export function General({ organization }: GeneralProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  usePageTitle(["Settings"]);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [changeManagementMessage, setChangeManagementMessage] = useState<string | null>(null);
  const [name, setName] = useState(organization.metadata?.name || "");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [showDeleteForm, setShowDeleteForm] = useState(false);
  const [changeManagementEnabled, setChangeManagementEnabled] = useState(
    organization.spec?.changeManagementEnabled ?? false,
  );
  /** When false, pending email invites accept any configured OAuth provider ([] server-side). */
  const [oauthRestrictProviders, setOauthRestrictProviders] = useState(false);
  const [oauthGithub, setOauthGithub] = useState(true);
  const [oauthGoogle, setOauthGoogle] = useState(true);
  const [oauthMessage, setOauthMessage] = useState<string | null>(null);
  const [oauthSelectionError, setOauthSelectionError] = useState<string | null>(null);
  const [allowDirectEmailInviteCompletion, setAllowDirectEmailInviteCompletion] = useState(
    organization.spec?.allowDirectEmailInviteCompletion ?? true,
  );
  const [directEmailInviteMessage, setDirectEmailInviteMessage] = useState<string | null>(null);

  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const deleteOrganizationMutation = useDeleteOrganization(organizationId || "");
  const canUpdateOrg = canAct("org", "update");
  const canDeleteOrg = canAct("org", "delete");

  useEffect(() => {
    setChangeManagementEnabled(organization.spec?.changeManagementEnabled ?? false);
  }, [organization.spec?.changeManagementEnabled]);

  useEffect(() => {
    const d = oauthDraftFromAllowedProviders(organization.spec?.allowedOauthProviders?.providers);
    setOauthRestrictProviders(d.restrict);
    setOauthGithub(d.github);
    setOauthGoogle(d.google);
    setOauthSelectionError(null);
  }, [organization.spec?.allowedOauthProviders?.providers]);

  useEffect(() => {
    setAllowDirectEmailInviteCompletion(organization.spec?.allowDirectEmailInviteCompletion ?? true);
  }, [organization.spec?.allowDirectEmailInviteCompletion]);

  const handleSave = async () => {
    if (!canUpdateOrg) return;
    if (!organizationId) {
      console.error("Organization ID is missing");
      return;
    }

    try {
      setSaveMessage(null);

      await updateOrganizationMutation.mutateAsync({
        name: name,
      });

      setSaveMessage("Organization updated successfully");
      setTimeout(() => setSaveMessage(null), 3000);
    } catch {
      setSaveMessage("Failed to update organization");
      setTimeout(() => setSaveMessage(null), 3000);
    }
  };

  const handleDelete = async () => {
    if (!canDeleteOrg) return;
    if (!organizationId) {
      console.error("Organization ID is missing");
      return;
    }
    if (deleteConfirmation !== (organization.metadata?.name || "")) {
      setDeleteError("Confirmation text does not match the organization name.");
      return;
    }

    try {
      setDeleteError(null);
      await deleteOrganizationMutation.mutateAsync();
      window.location.href = "/";
    } catch {
      setDeleteError("Failed to delete organization. Please try again.");
    }
  };

  const savedOauthFromServer = organization.spec?.allowedOauthProviders?.providers;
  const oauthSavedPolicySummary = (() => {
    const saved = savedOauthFromServer ?? [];
    if (saved.length === 0) {
      return "Saved: any OAuth provider can complete pending email invitations.";
    }
    const labels = saved.map((p) => (p === "github" ? "GitHub" : p === "google" ? "Google" : p));
    const joined = labels.length === 2 ? `${labels[0]} and ${labels[1]}` : labels.join(", ");
    return `Saved: only ${joined} can complete pending email invitations.`;
  })();

  const oauthProvidersToSave = (): string[] => {
    if (!oauthRestrictProviders) {
      return [];
    }
    const out: string[] = [];
    if (oauthGithub) {
      out.push("github");
    }
    if (oauthGoogle) {
      out.push("google");
    }
    return out;
  };

  const oauthProvidersListEqual = (a: string[], b: string[]) => {
    if (a.length !== b.length) {
      return false;
    }
    const sortedA = [...a].sort();
    const sortedB = [...b].sort();
    return sortedA.every((v, i) => v === sortedB[i]);
  };

  const serverOauthList = organization.spec?.allowedOauthProviders?.providers ?? [];
  const draftOauthList = oauthProvidersToSave();
  const oauthHasUnsavedChanges = !oauthProvidersListEqual(serverOauthList, draftOauthList);

  const handleSaveOAuthProviders = async () => {
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
      await updateOrganizationMutation.mutateAsync({
        allowedOauthProviders: oauthProvidersToSave(),
      });
      setOauthMessage("OAuth invitation settings saved");
      setTimeout(() => setOauthMessage(null), 3000);
    } catch {
      setOauthMessage("Failed to save OAuth invitation settings");
      setTimeout(() => setOauthMessage(null), 3000);
    }
  };

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

  const handleChangeManagementToggle = async (enabled: boolean) => {
    if (!canUpdateOrg || !organizationId) {
      return;
    }

    const previous = changeManagementEnabled;
    setChangeManagementEnabled(enabled);
    setChangeManagementMessage(null);

    try {
      await updateOrganizationMutation.mutateAsync({
        changeManagementEnabled: enabled,
      });
      setChangeManagementMessage(`Change management ${enabled ? "enabled" : "disabled"}`);
      setTimeout(() => setChangeManagementMessage(null), 3000);
    } catch {
      setChangeManagementEnabled(previous);
      setChangeManagementMessage("Failed to update change management");
      setTimeout(() => setChangeManagementMessage(null), 3000);
    }
  };

  return (
    <div className="space-y-6 pt-6 text-left">
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
        <Field className="space-y-4">
          <Label
            htmlFor="organization-name-input"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
          >
            Organization Name
          </Label>
          <Input
            id="organization-name-input"
            type="text"
            value={name}
            onChange={(e: ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
            className="max-w-sm"
            disabled={!canUpdateOrg}
          />
          <div className="flex items-center gap-4">
            <PermissionTooltip
              allowed={canUpdateOrg || permissionsLoading}
              message="You don't have permission to update this organization."
            >
              <LoadingButton
                type="button"
                onClick={handleSave}
                disabled={!canUpdateOrg}
                loading={updateOrganizationMutation.isPending}
                loadingText="Saving..."
                className="max-w-48"
              >
                Save Changes
              </LoadingButton>
            </PermissionTooltip>
            {saveMessage && (
              <span className={`text-sm ${saveMessage.includes("successfully") ? "text-green-600" : "text-red-600"}`}>
                {saveMessage}
              </span>
            )}
          </div>
        </Field>
      </Fieldset>

      <PermissionTooltip
        allowed={canUpdateOrg || permissionsLoading}
        message="You don't have permission to update this organization."
        className="w-full"
      >
        <Fieldset
          className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6"
          data-testid="oauth-invitation-settings-card"
        >
          <div className="flex items-start justify-between gap-6">
            <div>
              <Label
                htmlFor="organization-oauth-invite-restrict-switch"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
              >
                OAuth providers for email invitations
              </Label>
              <p className="text-sm text-gray-500 dark:text-gray-400 max-w-prose">
                Applies when someone signs in with GitHub or Google and has a pending email invitation to this
                organization. Turn the switch on to allow only the providers you select; turn it off to allow any
                configured OAuth provider.
              </p>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-prose">
                Continue with email and password sign-in (when your installation allows them) are configured in the
                section below.
              </p>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">{oauthSavedPolicySummary}</p>
              {oauthHasUnsavedChanges ? (
                <p className="text-xs text-amber-800 dark:text-amber-400 mt-2">
                  You have unsaved changes. Click Save OAuth settings to apply them.
                </p>
              ) : null}
            </div>
            <div className="flex items-center gap-3 shrink-0">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {oauthRestrictProviders ? "Restricted" : "Any provider"}
              </span>
              <Switch
                id="organization-oauth-invite-restrict-switch"
                checked={oauthRestrictProviders}
                onCheckedChange={(checked: boolean) => {
                  setOauthRestrictProviders(checked);
                  setOauthSelectionError(null);
                  if (checked && !oauthGithub && !oauthGoogle) {
                    setOauthGithub(true);
                    setOauthGoogle(true);
                  }
                }}
                disabled={updateOrganizationMutation.isPending || !canUpdateOrg}
                aria-label="Restrict OAuth providers for pending email invitations"
              />
            </div>
          </div>

          {oauthRestrictProviders ? (
            <div className="mt-4 border-t border-gray-200 pt-4 dark:border-gray-700">
              <p className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Allowed OAuth providers</p>
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-6">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="org-oauth-github"
                    checked={oauthGithub}
                    onCheckedChange={(checked: boolean | "indeterminate") => {
                      setOauthGithub(checked === true);
                      setOauthSelectionError(null);
                    }}
                    disabled={!canUpdateOrg || updateOrganizationMutation.isPending}
                  />
                  <Label htmlFor="org-oauth-github" className="text-sm font-normal cursor-pointer">
                    GitHub
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="org-oauth-google"
                    checked={oauthGoogle}
                    onCheckedChange={(checked: boolean | "indeterminate") => {
                      setOauthGoogle(checked === true);
                      setOauthSelectionError(null);
                    }}
                    disabled={!canUpdateOrg || updateOrganizationMutation.isPending}
                  />
                  <Label htmlFor="org-oauth-google" className="text-sm font-normal cursor-pointer">
                    Google
                  </Label>
                </div>
              </div>
            </div>
          ) : null}

          {oauthSelectionError ? <p className="mt-3 text-sm text-red-600">{oauthSelectionError}</p> : null}

          <div className="mt-4 flex items-center gap-4">
            <LoadingButton
              type="button"
              onClick={handleSaveOAuthProviders}
              disabled={!canUpdateOrg}
              loading={updateOrganizationMutation.isPending}
              loadingText="Saving..."
              className="max-w-48"
            >
              Save OAuth settings
            </LoadingButton>
            {oauthMessage ? (
              <span className={`text-sm ${oauthMessage.includes("Failed") ? "text-red-600" : "text-green-600"}`}>
                {oauthMessage}
              </span>
            ) : null}
          </div>
        </Fieldset>
      </PermissionTooltip>

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
                <strong>Continue with email</strong> button on the login page (you get a sign-in link or a code by
                email), or email plus password when password login is enabled for your installation.
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

      {isChangeManagementSettingsEnabled() ? (
        <PermissionTooltip
          allowed={canUpdateOrg || permissionsLoading}
          message="You don't have permission to update this organization."
          className="w-full"
        >
          <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
            <div className="flex items-start justify-between gap-6">
              <div>
                <Label
                  htmlFor="organization-change-management-switch"
                  className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                >
                  Change Management
                </Label>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Require change requests with approvals before publishing canvas changes. When enabled at the
                  organization level, change management is enforced for every canvas and cannot be turned off per
                  canvas.
                </p>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  When disabled here, each canvas can choose its own change management setting. New canvases inherit
                  this organization setting by default.
                </p>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {changeManagementEnabled ? "Enabled" : "Disabled"}
                </span>
                <Switch
                  id="organization-change-management-switch"
                  checked={changeManagementEnabled}
                  onCheckedChange={handleChangeManagementToggle}
                  disabled={updateOrganizationMutation.isPending || !canUpdateOrg}
                  aria-label="Toggle change management"
                />
              </div>
            </div>
            {changeManagementMessage ? (
              <p
                className={`mt-3 text-sm ${changeManagementMessage.includes("Failed") ? "text-red-600" : "text-green-600"}`}
              >
                {changeManagementMessage}
              </p>
            ) : null}
          </Fieldset>
        </PermissionTooltip>
      ) : null}

      <Fieldset className="bg-white border border-gray-300 rounded-lg p-6 space-y-4">
        {!showDeleteForm ? (
          <PermissionTooltip
            allowed={canDeleteOrg || permissionsLoading}
            message="You don't have permission to delete this organization."
          >
            <button
              type="button"
              onClick={() => {
                if (!canDeleteOrg) return;
                setShowDeleteForm(true);
              }}
              className="flex items-center gap-2 text-sm text-gray-800 hover:text-red-500"
              disabled={!canDeleteOrg}
            >
              <Trash2 className="h-4 w-4" />
              Delete Organization...
            </button>
          </PermissionTooltip>
        ) : (
          <>
            <div>
              <Heading level={3} className="!text-lg text-red-500 dark:text-red-400">
                Delete Organization
              </Heading>
              <p className="text-sm max-w-prose text-gray-800 dark:text-red-300 mt-2 mb-6">
                Deleting your organization is permanent and will remove all canvases, members, and settings. This action
                cannot be undone.
              </p>
            </div>
            <Field>
              <Label
                htmlFor="delete-organization-confirmation-input"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
              >
                Type "{organization.metadata?.name}" to confirm
              </Label>
              <Input
                id="delete-organization-confirmation-input"
                type="text"
                value={deleteConfirmation}
                onChange={(e: ChangeEvent<HTMLInputElement>) => setDeleteConfirmation(e.target.value)}
                placeholder={organization.metadata?.name || "Organization name"}
                className="max-w-sm"
                disabled={!canDeleteOrg}
              />
            </Field>
            <div className="flex items-center gap-4">
              <PermissionTooltip
                allowed={canDeleteOrg || permissionsLoading}
                message="You don't have permission to delete this organization."
              >
                <LoadingButton
                  type="button"
                  variant="outline"
                  onClick={handleDelete}
                  disabled={
                    deleteConfirmation !== (organization.metadata?.name || "") || !organizationId || !canDeleteOrg
                  }
                  loading={deleteOrganizationMutation.isPending}
                  loadingText="Deleting..."
                  className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400 gap-1"
                >
                  <Trash2 className="h-4 w-4" />
                  Delete Organization
                </LoadingButton>
              </PermissionTooltip>
              {deleteError && <span className="text-sm text-red-600">{deleteError}</span>}
            </div>
          </>
        )}
      </Fieldset>
    </div>
  );
}
