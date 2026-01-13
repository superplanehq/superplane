import { useState } from "react";
import { Trash2 } from "lucide-react";
import { useParams } from "react-router-dom";
import type { OrganizationsOrganization } from "../../../api-client/types.gen";
import { Field, Fieldset, Label } from "../../../components/Fieldset/fieldset";
import { Heading } from "../../../components/Heading/heading";
import { Input } from "../../../components/Input/input";
import { useDeleteOrganization, useUpdateOrganization } from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";

interface GeneralProps {
  organization: OrganizationsOrganization;
}

export function General({ organization }: GeneralProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [name, setName] = useState(organization.metadata?.name || "");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [showDeleteForm, setShowDeleteForm] = useState(false);

  // Use React Query mutation hook
  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const deleteOrganizationMutation = useDeleteOrganization(organizationId || "");

  const handleSave = async () => {
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
    } catch (err) {
      setSaveMessage("Failed to update organization");
      console.error("Error updating organization:", err);
      setTimeout(() => setSaveMessage(null), 3000);
    }
  };

  const handleDelete = async () => {
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
    } catch (err) {
      console.error("Failed to delete organization:", err);
      setDeleteError("Failed to delete organization. Please try again.");
    }
  };
  return (
    <div className="space-y-6 pt-6 text-left">
      <Fieldset className="bg-white dark:bg-gray-950 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
        <Field className="space-y-4">
          <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Organization Name</Label>
          <Input type="text" value={name} onChange={(e) => setName(e.target.value)} className="max-w-sm" />
          <div className="flex items-center gap-4">
            <Button
              type="button"
              onClick={handleSave}
              disabled={updateOrganizationMutation.isPending}
              className="max-w-48"
            >
              {updateOrganizationMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
            {saveMessage && (
              <span className={`text-sm ${saveMessage.includes("successfully") ? "text-green-600" : "text-red-600"}`}>
                {saveMessage}
              </span>
            )}
          </div>
        </Field>
      </Fieldset>

      <Fieldset className="bg-white border border-gray-300 rounded-lg p-6 space-y-4">
        {!showDeleteForm ? (
          <button
            type="button"
            onClick={() => setShowDeleteForm(true)}
            className="flex items-center gap-2 text-sm text-gray-800 hover:text-red-500"
          >
            <Trash2 className="h-4 w-4" />
            Delete Organization...
          </button>
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
              <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Type "{organization.metadata?.name}" to confirm
              </Label>
              <Input
                type="text"
                value={deleteConfirmation}
                onChange={(e) => setDeleteConfirmation(e.target.value)}
                placeholder={organization.metadata?.name || "Organization name"}
                className="max-w-sm"
              />
            </Field>
            <div className="flex items-center gap-4">
              <Button
                type="button"
                variant="outline"
                onClick={handleDelete}
                disabled={
                  deleteOrganizationMutation.isPending ||
                  deleteConfirmation !== (organization.metadata?.name || "") ||
                  !organizationId
                }
                className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400 gap-1"
              >
                <Trash2 className="h-4 w-4" />
                {deleteOrganizationMutation.isPending ? "Deleting..." : "Delete Organization"}
              </Button>
              {deleteError && <span className="text-sm text-red-600">{deleteError}</span>}
            </div>
          </>
        )}
      </Fieldset>
    </div>
  );
}
