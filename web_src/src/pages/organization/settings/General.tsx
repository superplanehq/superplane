import { useState } from "react";
import { Trash2 } from "lucide-react";
import { useParams } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import type { OrganizationsOrganization } from "../../../api-client/types.gen";
import { Field, Fieldset, Label } from "../../../components/Fieldset/fieldset";
import { Heading } from "../../../components/Heading/heading";
import { Input } from "../../../components/Input/input";
import { useDeleteOrganization, useUpdateOrganization } from "../../../hooks/useOrganizationData";
import { LoadingButton } from "@/components/ui/loading-button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/usePermissions";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { settingsCardClassName } from "./settingsPageStyles";

interface GeneralProps {
  organization: OrganizationsOrganization;
}

export function General({ organization }: GeneralProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  usePageTitle(["Settings"]);
  useReportPageReady(!permissionsLoading);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [name, setName] = useState(organization.metadata?.name || "");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [showDeleteForm, setShowDeleteForm] = useState(false);

  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const deleteOrganizationMutation = useDeleteOrganization(organizationId || "");
  const canUpdateOrg = canAct("org", "update");
  const canDeleteOrg = canAct("org", "delete");

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

  return (
    <div className="space-y-6 pt-6 text-left">
      <Fieldset className={cn("space-y-6", settingsCardClassName)}>
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
            onChange={(e) => setName(e.target.value)}
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
              <span
                className={`text-sm ${saveMessage.includes("successfully") ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}`}
              >
                {saveMessage}
              </span>
            )}
          </div>
        </Field>
      </Fieldset>

      <Fieldset className={cn("space-y-4", settingsCardClassName)}>
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
              className="flex items-center gap-2 text-sm text-gray-800 hover:text-red-500 dark:text-gray-100 dark:hover:text-red-400"
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
                onChange={(e) => setDeleteConfirmation(e.target.value)}
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
                  onClick={handleDelete}
                  disabled={
                    deleteConfirmation !== (organization.metadata?.name || "") || !organizationId || !canDeleteOrg
                  }
                  loading={deleteOrganizationMutation.isPending}
                  loadingText="Deleting..."
                  className={cn(appDarkModeClasses.destructiveSoftAction, "gap-1")}
                >
                  <Trash2 className="h-4 w-4" />
                  Delete Organization
                </LoadingButton>
              </PermissionTooltip>
              {deleteError && <span className="text-sm text-red-600 dark:text-red-400">{deleteError}</span>}
            </div>
          </>
        )}
      </Fieldset>
    </div>
  );
}
