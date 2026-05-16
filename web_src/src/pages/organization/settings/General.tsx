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
import { usePermissions } from "@/contexts/usePermissions";
import { isChangeManagementSettingsEnabled } from "@/lib/env";
import { DirectEmailInviteSettingsCard } from "./components/DirectEmailInviteSettingsCard";
import { OAuthInvitationSettingsCard } from "./components/OAuthInvitationSettingsCard";

interface GeneralProps {
  organization: OrganizationsOrganization;
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

  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const updateChangeManagementMutation = useUpdateOrganization(organizationId || "");
  const deleteOrganizationMutation = useDeleteOrganization(organizationId || "");
  const canUpdateOrg = canAct("org", "update");
  const canDeleteOrg = canAct("org", "delete");

  useEffect(() => {
    setChangeManagementEnabled(organization.spec?.changeManagementEnabled ?? false);
  }, [organization.spec?.changeManagementEnabled]);

  const handleSave = async () => {
    if (!canUpdateOrg) return;
    if (!organizationId) {
      console.error("Organization ID is missing");
      return;
    }

    try {
      setSaveMessage(null);
      await updateOrganizationMutation.mutateAsync({ name: name });
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

  const handleChangeManagementToggle = async (enabled: boolean) => {
    if (!canUpdateOrg || !organizationId) {
      return;
    }

    const previous = changeManagementEnabled;
    setChangeManagementEnabled(enabled);
    setChangeManagementMessage(null);

    try {
      await updateChangeManagementMutation.mutateAsync({
        changeManagementEnabled: enabled,
      });
      if (enabled) {
        setChangeManagementMessage("Change management enabled");
        setTimeout(() => setChangeManagementMessage(null), 3000);
      }
    } catch {
      setChangeManagementEnabled(previous);
      setChangeManagementMessage("Failed to update change management");
      setTimeout(() => setChangeManagementMessage(null), 3000);
    }
  };

  const orgId = organizationId || "";

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

      <OAuthInvitationSettingsCard
        organization={organization}
        organizationId={orgId}
        canUpdateOrg={canUpdateOrg}
        permissionsLoading={permissionsLoading}
        updateOrganizationMutation={updateOrganizationMutation}
      />

      <DirectEmailInviteSettingsCard
        organization={organization}
        organizationId={orgId}
        canUpdateOrg={canUpdateOrg}
        permissionsLoading={permissionsLoading}
        updateOrganizationMutation={updateOrganizationMutation}
      />

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
                  disabled={updateChangeManagementMutation.isPending || !canUpdateOrg}
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
