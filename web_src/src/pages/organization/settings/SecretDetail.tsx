import { Breadcrumbs } from "@/components/Breadcrumbs/breadcrumbs";
import { Textarea } from "@/components/Textarea/textarea";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { Edit2, Key, Loader2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  useDeleteSecret,
  useSecret,
  useUpdateSecret,
  type UpdateSecretParams,
} from "@/hooks/useSecrets";

const MASKED_VALUE_PLACEHOLDER = "***";

interface SecretDetailProps {
  organizationId: string;
}

export function SecretDetail({ organizationId }: SecretDetailProps) {
  const navigate = useNavigate();
  const { secretId } = useParams<{ secretId: string }>();
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editingKeyName, setEditingKeyName] = useState("");
  const [editingValue, setEditingValue] = useState("");
  const [isAddingKey, setIsAddingKey] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  const { data: secret, isLoading, error } = useSecret(
    organizationId,
    "DOMAIN_TYPE_ORGANIZATION",
    secretId || "",
  );
  const updateSecretMutation = useUpdateSecret(
    organizationId,
    "DOMAIN_TYPE_ORGANIZATION",
    secretId || "",
  );
  const deleteSecretMutation = useDeleteSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  const handleSaveEdit = async () => {
    if (!secret || !editingKey || !editingValue.trim()) return;
    const newName = editingKeyName.trim();
    if (!newName) {
      showErrorToast("Key name is required");
      return;
    }
    const secretData = secret.spec?.local?.data || {};
    const otherKeys = Object.keys(secretData).filter((k) => k !== editingKey);
    if (newName !== editingKey && otherKeys.includes(newName)) {
      showErrorToast("Key already exists");
      return;
    }
    const envVars = [
      ...otherKeys.map((k) => ({ name: k, value: MASKED_VALUE_PLACEHOLDER })),
      { name: newName, value: editingValue.trim() },
    ];
    try {
      const params: UpdateSecretParams = {
        name: secret.metadata?.name || "",
        environmentVariables: envVars,
        provider: secret.spec?.provider ?? "PROVIDER_LOCAL",
        secretId: secret.metadata?.id,
      };
      await updateSecretMutation.mutateAsync(params);
      showSuccessToast(newName !== editingKey ? "Key updated" : "Value updated");
      setEditingKey(null);
      setEditingKeyName("");
      setEditingValue("");
    } catch (err) {
      showErrorToast(`Failed to update: ${getApiErrorMessage(err)}`);
    }
  };

  const handleCancelEdit = () => {
    setEditingKey(null);
    setEditingKeyName("");
    setEditingValue("");
  };

  const handleAddKey = async () => {
    if (!secret || !newKey.trim() || !newValue.trim()) {
      showErrorToast("Key and value are required");
      return;
    }
    const secretData = secret.spec?.local?.data || {};
    if (Object.keys(secretData).includes(newKey.trim())) {
      showErrorToast("Key already exists");
      return;
    }
    const envVars = [
      ...Object.entries(secretData).map(([k]) => ({ name: k, value: MASKED_VALUE_PLACEHOLDER })),
      { name: newKey.trim(), value: newValue.trim() },
    ];
    try {
      const params: UpdateSecretParams = {
        name: secret.metadata?.name || "",
        environmentVariables: envVars,
        provider: secret.spec?.provider ?? "PROVIDER_LOCAL",
        secretId: secret.metadata?.id,
      };
      await updateSecretMutation.mutateAsync(params);
      showSuccessToast("Key added");
      setIsAddingKey(false);
      setNewKey("");
      setNewValue("");
    } catch (err) {
      showErrorToast(`Failed to add key: ${getApiErrorMessage(err)}`);
    }
  };

  const handleCancelAdd = () => {
    setIsAddingKey(false);
    setNewKey("");
    setNewValue("");
  };

  const handleRemoveKey = async (keyToRemove: string) => {
    if (!secret) return;
    const secretData = secret.spec?.local?.data || {};
    const remaining = Object.entries(secretData)
      .filter(([k]) => k !== keyToRemove)
      .map(([k]) => ({ name: k, value: MASKED_VALUE_PLACEHOLDER }));
    if (remaining.length === 0) {
      showErrorToast("Secret must have at least one key");
      return;
    }
    try {
      const params: UpdateSecretParams = {
        name: secret.metadata?.name || "",
        environmentVariables: remaining,
        provider: secret.spec?.provider ?? "PROVIDER_LOCAL",
        secretId: secret.metadata?.id,
      };
      await updateSecretMutation.mutateAsync(params);
      showSuccessToast("Key removed");
      if (editingKey === keyToRemove) {
        setEditingKey(null);
        setEditingValue("");
      }
    } catch (err) {
      showErrorToast(`Failed to remove key: ${getApiErrorMessage(err)}`);
    }
  };

  const handleDelete = async () => {
    if (!secret) return;
    if (
      !confirm(
        `Are you sure you want to delete the secret "${secret.metadata?.name}"? This action cannot be undone.`,
      )
    ) {
      return;
    }
    try {
      await deleteSecretMutation.mutateAsync(secret.metadata?.id ?? "");
      showSuccessToast("Secret deleted successfully");
      navigate(`/${organizationId}/settings/secrets`);
    } catch (err) {
      showErrorToast(`Failed to delete secret: ${getApiErrorMessage(err)}`);
    }
  };

  const isUpdating = updateSecretMutation.isPending;
  const handleBackToSecrets = () => navigate(`/${organizationId}/settings/secrets`);

  if (isLoading || !secretId) {
    return (
      <div className="space-y-6 pt-6">
        <div className="mb-4">
          <Breadcrumbs
            items={[{ label: "Secrets", onClick: handleBackToSecrets }, { label: "Secret", current: true }]}
            showDivider={false}
          />
        </div>
        <div className="flex justify-center items-center h-32">
          <p className="text-gray-500 dark:text-gray-400">
            {!secretId ? "Secret not found" : "Loading..."}
          </p>
        </div>
      </div>
    );
  }

  if (error || !secret) {
    return (
      <div className="space-y-6 pt-6">
        <div className="mb-4">
          <Breadcrumbs
            items={[
              { label: "Secrets", onClick: handleBackToSecrets },
              { label: "Secret", current: true },
            ]}
            showDivider={false}
          />
        </div>
        <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded dark:border-red-800 dark:bg-red-900/20">
          <p>{error instanceof Error ? error.message : "Secret not found or failed to load."}</p>
        </div>
      </div>
    );
  }

  const secretData = secret.spec?.local?.data || {};
  const keys = Object.keys(secretData);

  return (
    <div className="space-y-6 pt-6">
      <div className="mb-4">
        <Breadcrumbs
          items={[
            { label: "Secrets", onClick: handleBackToSecrets },
            { label: secret.metadata?.name || "Unnamed Secret", current: true },
          ]}
          showDivider={false}
        />
      </div>

      <div className="bg-slate-50 dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <Key className="w-5 h-5 text-gray-500 dark:text-gray-400 shrink-0" />
            <h2 className="text-2xl font-semibold text-gray-800 dark:text-gray-100 truncate">
              {secret.metadata?.name || "Unnamed Secret"}
            </h2>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleDelete}
            disabled={deleteSecretMutation.isPending}
            className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 shrink-0"
            title="Delete secret"
            data-testid="secret-detail-delete"
          >
            {deleteSecretMutation.isPending ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
            <span className="ml-1">Delete secret</span>
          </Button>
        </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="p-6">
        <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100 mb-3">Key-value pairs</h3>
        <div className="space-y-2">
          {keys.map((keyName) => {
            const isEditing = editingKey === keyName;
            return (
              <div
                key={keyName}
                className={
                  isEditing
                    ? "flex gap-2 items-start rounded-md border border-gray-200 dark:border-gray-700 p-4 bg-gray-50/50 dark:bg-gray-800/50 space-y-2"
                    : "flex items-center gap-2 text-sm font-mono bg-gray-50 dark:bg-gray-800 rounded px-3 py-2"
                }
              >
                {isEditing ? (
                  <>
                    <div className="flex-1 min-w-0 space-y-2">
                      <Label className="text-xs text-gray-500 dark:text-gray-400 font-normal">Key</Label>
                      <Input
                        type="text"
                        value={editingKeyName}
                        onChange={(e) => setEditingKeyName(e.target.value)}
                        placeholder="Key name"
                        className="font-mono text-sm"
                        data-testid="secret-detail-edit-key-name"
                      />
                      <Label className="text-xs text-gray-500 dark:text-gray-400 font-normal block mt-2">
                        Value
                      </Label>
                      <Textarea
                        value={editingValue}
                        onChange={(e) => setEditingValue(e.target.value)}
                        placeholder="Value"
                        rows={8}
                        className="font-mono text-sm resize-y bg-white dark:bg-gray-900"
                        autoFocus
                        data-testid="secret-detail-edit-value"
                      />
                      <div className="flex gap-2 mt-2">
                        <Button
                          size="sm"
                          onClick={handleSaveEdit}
                          disabled={!editingKeyName.trim() || !editingValue.trim() || isUpdating}
                          data-testid="secret-detail-edit-save"
                        >
                          {isUpdating ? <Loader2 className="w-3 h-3 animate-spin" /> : "Save"}
                        </Button>
                        <Button variant="ghost" size="sm" onClick={handleCancelEdit}>
                          Cancel
                        </Button>
                      </div>
                    </div>
                  </>
                ) : (
                  <>
                    <span className="text-gray-700 dark:text-gray-200 shrink-0 w-32 truncate" title={keyName}>
                      {keyName}
                    </span>
                    <span className="text-gray-500 dark:text-gray-400">•••</span>
                    <Button
                      variant="ghost"
                      size="sm"
                        onClick={() => {
                          setEditingKey(keyName);
                          setEditingKeyName(keyName);
                          setEditingValue("");
                        }}
                      className="shrink-0 text-gray-600 dark:text-gray-300"
                      title="Edit value"
                      data-testid="secret-detail-edit-key"
                    >
                      <Edit2 className="w-3 h-3" />
                    </Button>
                    {keys.length > 1 && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveKey(keyName)}
                          disabled={isUpdating}
                          className="shrink-0 text-red-600 hover:text-red-700 dark:text-red-400"
                          title="Remove key"
                          data-testid="secret-detail-remove-key"
                        >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    )}
                  </>
                )}
              </div>
            );
          })}
          {isAddingKey && (
            <div className="flex gap-2 items-start rounded-md border border-gray-200 dark:border-gray-700 border-dashed p-4 bg-gray-50/50 dark:bg-gray-800/50">
              <div className="flex-1 min-w-0 space-y-2">
                <Label className="text-xs text-gray-500 dark:text-gray-400 font-normal">Key</Label>
                <Input
                  type="text"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  placeholder="Key name"
                  className="font-mono text-sm"
                  data-testid="secret-detail-add-key-name"
                />
                <Label className="text-xs text-gray-500 dark:text-gray-400 font-normal block mt-2">
                  Value
                </Label>
                <Textarea
                  value={newValue}
                  onChange={(e) => setNewValue(e.target.value)}
                  placeholder="Value"
                  rows={8}
                  className="font-mono text-sm resize-y bg-white dark:bg-gray-900"
                  data-testid="secret-detail-add-value"
                />
                <div className="flex gap-2 mt-2">
                  <Button
                    size="sm"
                    onClick={handleAddKey}
                    disabled={!newKey.trim() || !newValue.trim() || isUpdating}
                    data-testid="secret-detail-add-save"
                  >
                    {isUpdating ? <Loader2 className="w-3 h-3 animate-spin" /> : "Save"}
                  </Button>
                  <Button variant="ghost" size="sm" onClick={handleCancelAdd}>
                    Cancel
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
        {!isAddingKey && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setIsAddingKey(true);
              setNewKey("");
              setNewValue("");
            }}
            className="mt-3 text-xs"
            data-testid="secret-detail-add-key"
          >
            <Plus className="w-3 h-3 mr-1" />
            Add key
          </Button>
        )}
        </div>
      </div>
      </div>
    </div>
  );
}
