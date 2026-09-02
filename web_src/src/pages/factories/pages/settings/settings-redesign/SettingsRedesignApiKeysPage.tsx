import { useState } from "react";
import { Link } from "react-router";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useAPIKeys, useCreateAPIKey, useDeleteAPIKey } from "@/hooks/useApiKeys";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { useOrganizationSettingsPaths } from "@/lib/organizationSettingsPaths";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { KeyRound, Plus } from "lucide-react";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { SettingsListRow, SettingsStackedField } from "./settingsRedesignParts";

export function SettingsRedesignApiKeysPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: apiKeys = [], isLoading } = useAPIKeys(organizationId);
  const createMutation = useCreateAPIKey(organizationId);
  const deleteMutation = useDeleteAPIKey(organizationId);
  const settingsPaths = useOrganizationSettingsPaths(organizationId);
  const canCreate = canAct("api_keys", "create");
  const canDelete = canAct("api_keys", "delete");
  const [createOpen, setCreateOpen] = useState(false);
  const [newName, setNewName] = useState("");

  usePageTitle(["API keys"]);

  const handleCreate = async () => {
    if (!canCreate || !newName.trim()) return;
    try {
      await createMutation.mutateAsync({
        name: newName.trim(),
        description: "",
        role: "org_viewer",
        canvasIds: [],
      });
      showSuccessToast("API key created.");
      setCreateOpen(false);
      setNewName("");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create API key"));
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!canDelete) return;
    if (!window.confirm(`Delete API key ${name}? This cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(id);
      showSuccessToast("API key deleted.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to delete API key"));
    }
  };

  const sorted = [...apiKeys].sort((left, right) => (left.name || "").localeCompare(right.name || ""));

  return (
    <FactorySettingsPageFrame
      title="API keys"
      subtitle="Keys for programmatic access to SuperPlane."
      actions={
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You do not have permission to create API keys."
        >
          <Button
            type="button"
            size="sm"
            disabled={!canCreate}
            onClick={() => setCreateOpen(true)}
            data-testid="api-key-create-btn"
          >
            <Plus className="size-3.5" aria-hidden />
            Create API key
          </Button>
        </PermissionTooltip>
      }
    >
      {createOpen ? (
        <ApiKeyCreateForm
          name={newName}
          loading={createMutation.isPending}
          onNameChange={setNewName}
          onCancel={() => setCreateOpen(false)}
          onCreate={() => void handleCreate()}
        />
      ) : null}
      <ApiKeyList
        isLoading={isLoading}
        keys={sorted}
        canDelete={canDelete}
        permissionsLoading={permissionsLoading}
        deletePending={deleteMutation.isPending}
        detailPath={settingsPaths.apiKeyDetail}
        onDelete={handleDelete}
      />
    </FactorySettingsPageFrame>
  );
}

function ApiKeyCreateForm({
  name,
  loading,
  onNameChange,
  onCancel,
  onCreate,
}: {
  name: string;
  loading: boolean;
  onNameChange: (value: string) => void;
  onCancel: () => void;
  onCreate: () => void;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4" data-testid="settings-redesign-api-key-create">
      <SettingsStackedField htmlFor="settings-redesign-api-key-name" label="Name">
        <Input
          id="settings-redesign-api-key-name"
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
        />
      </SettingsStackedField>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <LoadingButton type="button" size="sm" disabled={!name.trim()} loading={loading} onClick={onCreate}>
          Create
        </LoadingButton>
      </div>
    </div>
  );
}

function ApiKeyList({
  isLoading,
  keys,
  canDelete,
  permissionsLoading,
  deletePending,
  detailPath,
  onDelete,
}: {
  isLoading: boolean;
  keys: Array<{ id?: string; name?: string; description?: string; hasToken?: boolean }>;
  canDelete: boolean;
  permissionsLoading: boolean;
  deletePending: boolean;
  detailPath: (id: string) => string;
  onDelete: (id: string, name: string) => Promise<void>;
}) {
  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading API keys...</p>;
  }
  if (keys.length === 0) {
    return <p className="text-[13px] text-muted-foreground">No API keys yet.</p>;
  }

  return (
    <ul data-testid="settings-redesign-api-keys">
      {keys.map((apiKey) => (
        <SettingsListRow
          key={apiKey.id}
          icon={<KeyRound className="size-4" aria-hidden />}
          title={
            <Link to={detailPath(apiKey.id || "")} data-testid="api-key-link">
              {apiKey.name || "Unnamed"}
            </Link>
          }
          subtitle={apiKey.description || "Organization-wide"}
          meta={apiKey.hasToken ? "Active" : "None"}
          action={
            <PermissionTooltip
              allowed={canDelete || permissionsLoading}
              message="You do not have permission to delete API keys."
            >
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={!canDelete || deletePending}
                onClick={() => void onDelete(apiKey.id || "", apiKey.name || "")}
                data-testid="api-key-delete-btn"
              >
                Delete
              </Button>
            </PermissionTooltip>
          }
        />
      ))}
    </ul>
  );
}
