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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import { getApiErrorMessage } from "@/utils/errors";

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
  const [agentApiKeyError, setAgentApiKeyError] = useState<string | null>(null);
  const [showAgentConfigureModal, setShowAgentConfigureModal] = useState(false);

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
  const openAIKeyConfigured = !!openAIKey?.configured;
  const isAgentModeZeroState = !openAIKeyConfigured && !agentModeEnabled;
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

  const handleSaveAgentOpenAIKey = async () => {
    if (!canUpdateOrg || !organizationId) return;
    if (!agentApiKey.trim()) {
      setAgentApiKeyError("OpenAI API key is required");
      return;
    }

    try {
      setAgentApiKeyError(null);
      const saveResult = await setAgentOpenAIKeyMutation.mutateAsync({
        apiKey: agentApiKey.trim(),
        validate: true,
      });
      const savedKeyStatus = saveResult?.agentSettings?.openaiKey?.status;

      if (savedKeyStatus === "invalid") {
        setAgentApiKeyError("Invalid OpenAI API key.");
        return;
      }

      if (isAgentModeZeroState) {
        await updateAgentSettingsMutation.mutateAsync(true);
      }
      setAgentApiKey("");
      setShowAgentConfigureModal(false);
    } catch (_err) {
      const apiError = getApiErrorMessage(_err);
      setAgentApiKeyError(apiError);
    }
  };

  const handleConfigureAgentMode = () => {
    if (!canUpdateOrg) return;
    setShowAgentConfigureModal(true);
  };

  const handleCancelConfigureAgentMode = () => {
    setAgentApiKey("");
    setAgentApiKeyError(null);
    setShowAgentConfigureModal(false);
  };

  const handleDisableAgentMode = async () => {
    if (!canUpdateOrg || !organizationId) return;

    try {
      await deleteAgentOpenAIKeyMutation.mutateAsync();
      await updateAgentSettingsMutation.mutateAsync(false);
      setAgentApiKey("");
      setAgentApiKeyError(null);
      setShowAgentConfigureModal(false);
    } catch (_err) {
      console.error(`Failed to disable Agent Mode: ${getApiErrorMessage(_err)}`);
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
        <Fieldset
          className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6"
          data-testid="agent-mode-settings-card"
        >
          <div className="flex items-start justify-between gap-6">
            <div>
              <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Agent Mode</Label>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {openAIKeyConfigured || agentModeEnabled
                  ? "Agent Mode is enabled."
                  : "Set up Agent Mode for this organization."}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {isAgentModeZeroState && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleConfigureAgentMode}
                  disabled={agentSettingsBusy || !canUpdateOrg}
                  data-testid="agent-mode-setup-button"
                >
                  Setup
                </Button>
              )}
              {!isAgentModeZeroState && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleConfigureAgentMode}
                  disabled={agentSettingsBusy || !canUpdateOrg}
                  data-testid="agent-mode-update-key-button"
                >
                  Update key
                </Button>
              )}
              {!isAgentModeZeroState && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleDisableAgentMode}
                  disabled={!canUpdateOrg || agentSettingsBusy || !openAIKeyConfigured}
                  data-testid="agent-mode-disable-button"
                >
                  Disable
                </Button>
              )}
            </div>
          </div>

          <Dialog open={showAgentConfigureModal} onOpenChange={setShowAgentConfigureModal}>
            <DialogContent showCloseButton={!agentSettingsBusy}>
              <DialogHeader>
                <DialogTitle>{isAgentModeZeroState ? "Set up Agent Mode" : "Configure Agent Mode"}</DialogTitle>
                <DialogDescription>
                  {isAgentModeZeroState ? "Add an OpenAI API key to set up Agent Mode." : "Update the OpenAI API key."}
                </DialogDescription>
              </DialogHeader>
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  handleSaveAgentOpenAIKey();
                }}
              >
                <div className="space-y-2">
                  <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300">OpenAI API key</Label>
                  <Input
                    type="password"
                    value={agentApiKey}
                    onChange={(e) => {
                      setAgentApiKey(e.target.value);
                      if (agentApiKeyError) {
                        setAgentApiKeyError(null);
                      }
                    }}
                    placeholder="sk-..."
                    disabled={!canUpdateOrg || agentSettingsBusy}
                    data-testid="agent-openai-key-input"
                    className={agentApiKeyError ? "border-red-300 focus-visible:ring-red-200" : undefined}
                    aria-invalid={agentApiKeyError ? "true" : "false"}
                  />
                  {agentApiKeyError && (
                    <p className="text-sm text-red-600 mt-1 whitespace-pre-line">{agentApiKeyError}</p>
                  )}
                </div>
                <DialogFooter className="mt-4">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCancelConfigureAgentMode}
                    disabled={agentSettingsBusy}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    disabled={!canUpdateOrg || agentSettingsBusy || !agentApiKey.trim()}
                    data-testid="agent-openai-key-save"
                  >
                    {setAgentOpenAIKeyMutation.isPending ? "Saving..." : "Save"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
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
