import { useState } from "react";
import { Trash2 } from "lucide-react";
import { useParams } from "react-router-dom";
import type { OrganizationsOrganization } from "../../../api-client/types.gen";
import { Field, Fieldset, Label } from "../../../components/Fieldset/fieldset";
import { Heading } from "../../../components/Heading/heading";
import { Input } from "../../../components/Input/input";
import {
  useDeleteOrganization,
  useUpdateOrganization,
  useOrganizationAgentSettings,
  useUpdateOrganizationAgentSettings,
  useSetOrganizationAgentOpenAIKey,
  useDeleteOrganizationAgentOpenAIKey,
} from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import { Switch } from "@/ui/switch";

interface GeneralProps {
  organization: OrganizationsOrganization;
}

export function General({ organization }: GeneralProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [name, setName] = useState(organization.metadata?.name || "");
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [showDeleteForm, setShowDeleteForm] = useState(false);
  const [agentApiKey, setAgentApiKey] = useState("");
  const [agentMessage, setAgentMessage] = useState<string | null>(null);
  const [agentError, setAgentError] = useState<string | null>(null);

  // Use React Query mutation hook
  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const deleteOrganizationMutation = useDeleteOrganization(organizationId || "");
  const { data: agentSettings, isLoading: loadingAgentSettings } = useOrganizationAgentSettings(organizationId || "");
  const updateAgentSettingsMutation = useUpdateOrganizationAgentSettings(organizationId || "");
  const setAgentOpenAIKeyMutation = useSetOrganizationAgentOpenAIKey(organizationId || "");
  const deleteAgentOpenAIKeyMutation = useDeleteOrganizationAgentOpenAIKey(organizationId || "");
  const canUpdateOrg = canAct("org", "update");
  const canDeleteOrg = canAct("org", "delete");

  const agentModeEnabled = agentSettings?.agentModeEnabled ?? false;
  const openAIKey = agentSettings?.openaiKey;
  const openAIKeyStatus = openAIKey?.status || "not_configured";
  const openAIKeyConfigured = !!openAIKey?.configured;
  const agentSettingsBusy =
    loadingAgentSettings ||
    updateAgentSettingsMutation.isPending ||
    setAgentOpenAIKeyMutation.isPending ||
    deleteAgentOpenAIKeyMutation.isPending;

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
    } catch (err) {
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
    } catch (err) {
      setDeleteError("Failed to delete organization. Please try again.");
    }
  };

  const handleAgentModeToggle = async (enabled: boolean) => {
    if (!canUpdateOrg || !organizationId) return;

    try {
      setAgentError(null);
      setAgentMessage(null);
      await updateAgentSettingsMutation.mutateAsync(enabled);
      setAgentMessage(enabled ? "Agent Mode enabled" : "Agent Mode disabled");
      setTimeout(() => setAgentMessage(null), 3000);
    } catch (_err) {
      setAgentError("Failed to update Agent Mode setting");
      setTimeout(() => setAgentError(null), 3000);
    }
  };

  const handleSaveAgentOpenAIKey = async () => {
    if (!canUpdateOrg || !organizationId) return;
    if (!agentApiKey.trim()) {
      setAgentError("OpenAI API key is required");
      return;
    }

    try {
      setAgentError(null);
      setAgentMessage(null);
      await setAgentOpenAIKeyMutation.mutateAsync({
        apiKey: agentApiKey.trim(),
        validate: true,
      });
      setAgentApiKey("");
      setAgentMessage("OpenAI key saved");
      setTimeout(() => setAgentMessage(null), 3000);
    } catch (_err) {
      setAgentError("Failed to save OpenAI key");
      setTimeout(() => setAgentError(null), 3000);
    }
  };

  const handleDeleteAgentOpenAIKey = async () => {
    if (!canUpdateOrg || !organizationId) return;

    try {
      setAgentError(null);
      setAgentMessage(null);
      await deleteAgentOpenAIKeyMutation.mutateAsync();
      setAgentMessage("OpenAI key removed");
      setTimeout(() => setAgentMessage(null), 3000);
    } catch (_err) {
      setAgentError("Failed to remove OpenAI key");
      setTimeout(() => setAgentError(null), 3000);
    }
  };

  const openAIKeyStatusDescription = () => {
    switch (openAIKeyStatus) {
      case "valid":
        return "Configured and validated.";
      case "invalid":
        return openAIKey?.validationError || "Configured, but key validation failed.";
      case "unchecked":
        return openAIKey?.validationError || "Configured, but validation is pending.";
      default:
        return "Not configured.";
    }
  };

  return (
    <div className="space-y-6 pt-6 text-left">
      <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
        <Field className="space-y-4">
          <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Organization Name</Label>
          <Input
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
              <Button
                type="button"
                onClick={handleSave}
                disabled={updateOrganizationMutation.isPending || !canUpdateOrg}
                className="max-w-48"
              >
                {updateOrganizationMutation.isPending ? "Saving..." : "Save Changes"}
              </Button>
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
        <Fieldset className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <div className="flex items-start justify-between gap-6">
            <div>
              <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Enable Agent Mode</Label>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Turn on Agent Mode for this organization. Add an OpenAI key to make Agent Mode effective.
              </p>
            </div>
            <Switch
              checked={agentModeEnabled}
              onCheckedChange={handleAgentModeToggle}
              disabled={agentSettingsBusy || !canUpdateOrg}
              aria-label="Enable Agent Mode"
            />
          </div>

          {agentError && (
            <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded mt-4">
              <p className="text-sm">{agentError}</p>
            </div>
          )}

          {agentMessage && (
            <div className="bg-white border border-green-300 text-green-600 px-4 py-2 rounded mt-4">
              <p className="text-sm">{agentMessage}</p>
            </div>
          )}

          {agentModeEnabled && (
            <div className="mt-4 space-y-4">
              <div>
                <p className="text-sm text-gray-700 dark:text-gray-300">
                  OpenAI key status:{" "}
                  <span className="font-medium text-gray-900 dark:text-gray-100">{openAIKeyStatus}</span>
                  {openAIKeyConfigured && openAIKey?.last4 ? (
                    <span className="text-gray-500 dark:text-gray-400"> (••••{openAIKey.last4})</span>
                  ) : null}
                </p>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{openAIKeyStatusDescription()}</p>
              </div>

              <div className="flex flex-wrap items-end gap-3">
                <div className="flex-1 min-w-[320px]">
                  <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    OpenAI API key
                  </Label>
                  <Input
                    type="password"
                    value={agentApiKey}
                    onChange={(e) => setAgentApiKey(e.target.value)}
                    placeholder="sk-..."
                    disabled={!canUpdateOrg || agentSettingsBusy}
                  />
                </div>
                <Button
                  type="button"
                  onClick={handleSaveAgentOpenAIKey}
                  disabled={!canUpdateOrg || agentSettingsBusy || !agentApiKey.trim()}
                >
                  {setAgentOpenAIKeyMutation.isPending ? "Saving..." : "Save key"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleDeleteAgentOpenAIKey}
                  disabled={!canUpdateOrg || agentSettingsBusy || !openAIKeyConfigured}
                >
                  {deleteAgentOpenAIKeyMutation.isPending ? "Removing..." : "Remove key"}
                </Button>
              </div>
            </div>
          )}
        </Fieldset>
      </PermissionTooltip>

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
              <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Type "{organization.metadata?.name}" to confirm
              </Label>
              <Input
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
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleDelete}
                  disabled={
                    deleteOrganizationMutation.isPending ||
                    deleteConfirmation !== (organization.metadata?.name || "") ||
                    !organizationId ||
                    !canDeleteOrg
                  }
                  className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400 gap-1"
                >
                  <Trash2 className="h-4 w-4" />
                  {deleteOrganizationMutation.isPending ? "Deleting..." : "Delete Organization"}
                </Button>
              </PermissionTooltip>
              {deleteError && <span className="text-sm text-red-600">{deleteError}</span>}
            </div>
          </>
        )}
      </Fieldset>
    </div>
  );
}
