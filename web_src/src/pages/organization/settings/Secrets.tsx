import { Icon } from "@/components/Icon";
import { Textarea } from "@/components/Textarea/textarea";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { Edit2, Eye, EyeOff, Key, Loader2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import type { SecretsSecret } from "../../../api-client/types.gen";
import {
  useCreateSecret,
  useDeleteSecret,
  useSecrets,
  useUpdateSecret,
  type CreateSecretParams,
  type UpdateSecretParams,
} from "../../../hooks/useSecrets";

interface SecretsProps {
  organizationId: string;
}

interface KeyValuePair {
  name: string;
  value: string;
}

export function Secrets({ organizationId }: SecretsProps) {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingSecret, setEditingSecret] = useState<SecretsSecret | null>(null);
  const [secretName, setSecretName] = useState("");
  const [keyValuePairs, setKeyValuePairs] = useState<KeyValuePair[]>([{ name: "", value: "" }]);
  const [visibleValues, setVisibleValues] = useState<Set<string>>(new Set());
  const [visibleEditValues, setVisibleEditValues] = useState<Set<number>>(new Set());

  const { data: secrets = [], isLoading } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const createSecretMutation = useCreateSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const deleteSecretMutation = useDeleteSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const updateSecretMutation = useUpdateSecret(
    organizationId,
    "DOMAIN_TYPE_ORGANIZATION",
    editingSecret?.metadata?.id || "",
  );

  const handleCreateClick = () => {
    setSecretName("");
    setKeyValuePairs([{ name: "", value: "" }]);
    setVisibleEditValues(new Set([0])); // Show first value by default
    setIsCreateModalOpen(true);
  };

  const handleEditClick = (secret: SecretsSecret) => {
    const secretData = secret.spec?.local?.data || {};
    const pairs = Object.entries(secretData).map(([name, value]) => ({ name, value }));
    setSecretName(secret.metadata?.name || "");
    setKeyValuePairs(pairs.length > 0 ? pairs : [{ name: "", value: "" }]);
    // Show all values by default when editing
    setVisibleEditValues(new Set(pairs.map((_, index) => index)));
    setEditingSecret(secret);
  };

  const handleCloseModal = () => {
    setIsCreateModalOpen(false);
    setEditingSecret(null);
    setSecretName("");
    setKeyValuePairs([{ name: "", value: "" }]);
    setVisibleEditValues(new Set());
    createSecretMutation.reset();
  };

  const handleCreate = async () => {
    if (!secretName?.trim()) {
      showErrorToast("Secret name is required");
      return;
    }

    const validPairs = keyValuePairs.filter((pair) => pair.name.trim() && pair.value.trim());
    if (validPairs.length === 0) {
      showErrorToast("At least one key-value pair is required");
      return;
    }

    // Check for duplicate keys
    const keys = validPairs.map((pair) => pair.name.trim());
    if (new Set(keys).size !== keys.length) {
      showErrorToast("Duplicate key names are not allowed");
      return;
    }

    try {
      const params: CreateSecretParams = {
        name: secretName.trim(),
        environmentVariables: validPairs.map((pair) => ({
          name: pair.name.trim(),
          value: pair.value.trim(),
        })),
      };
      await createSecretMutation.mutateAsync(params);
      showSuccessToast("Secret created successfully");
      handleCloseModal();
    } catch (error) {
      showErrorToast(`Failed to create secret: ${getApiErrorMessage(error)}`);
    }
  };

  const handleUpdate = async () => {
    if (!editingSecret || !secretName?.trim()) {
      showErrorToast("Secret name is required");
      return;
    }

    const validPairs = keyValuePairs.filter((pair) => pair.name.trim() && pair.value.trim());
    if (validPairs.length === 0) {
      showErrorToast("At least one key-value pair is required");
      return;
    }

    // Check for duplicate keys
    const keys = validPairs.map((pair) => pair.name.trim());
    if (new Set(keys).size !== keys.length) {
      showErrorToast("Duplicate key names are not allowed");
      return;
    }

    try {
      const params: UpdateSecretParams = {
        name: secretName.trim(),
        environmentVariables: validPairs.map((pair) => ({
          name: pair.name.trim(),
          value: pair.value.trim(),
        })),
      };
      await updateSecretMutation.mutateAsync(params);
      showSuccessToast("Secret updated successfully");
      handleCloseModal();
    } catch (error) {
      showErrorToast(`Failed to update secret: ${getApiErrorMessage(error)}`);
    }
  };

  const handleDelete = async (secret: SecretsSecret) => {
    if (!secret.metadata?.id) return;

    if (
      !confirm(`Are you sure you want to delete the secret "${secret.metadata.name}"? This action cannot be undone.`)
    ) {
      return;
    }

    try {
      await deleteSecretMutation.mutateAsync(secret.metadata.id);
      showSuccessToast("Secret deleted successfully");
    } catch (error) {
      showErrorToast(`Failed to delete secret: ${getApiErrorMessage(error)}`);
    }
  };

  const addKeyValuePair = () => {
    setKeyValuePairs([...keyValuePairs, { name: "", value: "" }]);
  };

  const removeKeyValuePair = (index: number) => {
    setKeyValuePairs(keyValuePairs.filter((_, i) => i !== index));
  };

  const updateKeyValuePair = (index: number, field: "name" | "value", value: string) => {
    const updated = [...keyValuePairs];
    updated[index] = { ...updated[index], [field]: value };
    setKeyValuePairs(updated);
  };

  const toggleValueVisibility = (secretId: string) => {
    const newVisible = new Set(visibleValues);
    if (newVisible.has(secretId)) {
      newVisible.delete(secretId);
    } else {
      newVisible.add(secretId);
    }
    setVisibleValues(newVisible);
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex justify-center items-center h-32">
          <p className="text-gray-500 dark:text-gray-400">Loading secrets...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h2 className="text-lg font-medium mb-2">Secrets</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Manage key-value pairs and sensitive data for your organization.
          </p>
        </div>
        {secrets.length > 0 && (
          <Button color="blue" onClick={handleCreateClick} className="flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Create Secret
          </Button>
        )}
      </div>

      {secrets.length === 0 ? (
        <div className="text-center py-12 bg-white border border-gray-300 dark:border-gray-700 rounded-md">
          <Key className="w-8 h-8 text-gray-400 mx-auto mb-2" />
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">No secrets found</p>
          <Button color="blue" onClick={handleCreateClick} className="flex items-center gap-2 mx-auto">
            <Plus className="w-4 h-4" />
            Create your first secret
          </Button>
        </div>
      ) : (
        <div className="space-y-4">
          {secrets
            .sort((a, b) => (a.metadata?.name || "").localeCompare(b.metadata?.name || ""))
            .map((secret) => {
              const secretId = secret.metadata?.id || "";
              const secretData = secret.spec?.local?.data || {};
              const isVisible = visibleValues.has(secretId);
              const envVarCount = Object.keys(secretData).length;

              return (
                <div key={secretId} className="bg-white border border-gray-300 dark:border-gray-700 rounded-md p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Key className="w-4 h-4 text-gray-500 dark:text-gray-400" />
                        <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                          {secret.metadata?.name || "Unnamed Secret"}
                        </h3>
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
                        {envVarCount} key-value pair{envVarCount !== 1 ? "s" : ""}
                      </p>
                      {isVisible && (
                        <div className="space-y-2 mt-3">
                          {Object.entries(secretData).map(([key, value]) => (
                            <div key={key} className="text-xs font-mono bg-gray-50 dark:bg-gray-800 rounded p-2">
                              <span className="text-gray-600 dark:text-gray-400">{key}:</span>{" "}
                              <span className="text-gray-800 dark:text-gray-200">{value}</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="flex items-start gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => toggleValueVisibility(secretId)}
                        className="text-xs"
                        title={isVisible ? "Hide values" : "Show values"}
                      >
                        {isVisible ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleEditClick(secret)}
                        className="text-xs"
                        title="Edit secret"
                      >
                        <Edit2 className="w-3 h-3" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDelete(secret)}
                        className="text-xs text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                        title="Delete secret"
                        disabled={deleteSecretMutation.isPending}
                      >
                        {deleteSecretMutation.isPending ? (
                          <Loader2 className="w-3 h-3 animate-spin" />
                        ) : (
                          <Trash2 className="w-3 h-3" />
                        )}
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
        </div>
      )}

      {/* Create/Edit Modal */}
      {(isCreateModalOpen || editingSecret) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <Key className="w-6 h-6 text-gray-500 dark:text-gray-400" />
                  <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                    {editingSecret ? "Edit Secret" : "Create Secret"}
                  </h3>
                </div>
                <button
                  onClick={handleCloseModal}
                  className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                  disabled={createSecretMutation.isPending || updateSecretMutation.isPending}
                >
                  <Icon name="x" size="sm" />
                </button>
              </div>

              <div className="space-y-4">
                {/* Secret Name Field */}
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">
                    Secret Name
                    <span className="text-red-500 ml-1">*</span>
                  </Label>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">A unique name for this secret</p>
                  <Input
                    type="text"
                    value={secretName}
                    onChange={(e) => setSecretName(e.target.value)}
                    placeholder="e.g., production-api-keys"
                    required
                  />
                </div>

                {/* Key-Value Pairs */}
                <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
                  <div className="mb-4">
                    <Label className="text-gray-800 dark:text-gray-100">
                      Key-Value Pairs
                      <span className="text-red-500 ml-1">*</span>
                    </Label>
                  </div>
                  <div className="space-y-3">
                    {keyValuePairs.map((pair, index) => {
                      return (
                        <div key={index} className="flex gap-2 items-start">
                          <div className="flex-1">
                            <Input
                              type="text"
                              value={pair.name}
                              onChange={(e) => updateKeyValuePair(index, "name", e.target.value)}
                              placeholder="Key"
                              className="mb-2"
                            />
                            <Textarea
                              value={pair.value}
                              onChange={(e) => updateKeyValuePair(index, "value", e.target.value)}
                              placeholder="Value"
                              rows={4}
                              className="font-mono text-sm"
                            />
                          </div>
                          {keyValuePairs.length > 1 && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => removeKeyValuePair(index)}
                              className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 mt-0"
                              title="Remove pair"
                            >
                              <Trash2 className="w-3 h-3" />
                            </Button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                  <div className="mt-3">
                    <Button variant="outline" size="sm" onClick={addKeyValuePair} className="text-xs">
                      <Plus className="w-3 h-3 mr-1" />
                      Add Pair
                    </Button>
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    Add key-value pairs that will be stored securely
                  </p>
                </div>
              </div>

              <div className="flex justify-start gap-3 mt-6">
                <Button
                  color="blue"
                  onClick={editingSecret ? handleUpdate : handleCreate}
                  disabled={createSecretMutation.isPending || updateSecretMutation.isPending || !secretName?.trim()}
                  className="flex items-center gap-2"
                >
                  {createSecretMutation.isPending || updateSecretMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      {editingSecret ? "Updating..." : "Creating..."}
                    </>
                  ) : editingSecret ? (
                    "Update Secret"
                  ) : (
                    "Create Secret"
                  )}
                </Button>
                <Button
                  variant="outline"
                  onClick={handleCloseModal}
                  disabled={createSecretMutation.isPending || updateSecretMutation.isPending}
                >
                  Cancel
                </Button>
              </div>

              {(createSecretMutation.isError || updateSecretMutation.isError) && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    {editingSecret ? "Failed to update secret" : "Failed to create secret"}:{" "}
                    {getApiErrorMessage(createSecretMutation.error || updateSecretMutation.error)}
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
