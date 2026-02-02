import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Link } from "@/components/Link/link";
import { Textarea } from "@/components/Textarea/textarea";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { usePermissions } from "@/contexts/PermissionsContext";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { Key, Loader2, Plus, Trash2 } from "lucide-react";
import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  useCreateSecret,
  useSecrets,
  type CreateSecretParams,
} from "@/hooks/useSecrets";

interface SecretsProps {
  organizationId: string;
}

interface KeyValuePair {
  name: string;
  value: string;
}

export function Secrets({ organizationId }: SecretsProps) {
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [secretName, setSecretName] = useState("");
  const [keyValuePairs, setKeyValuePairs] = useState<KeyValuePair[]>([{ name: "", value: "" }]);
  const canCreateSecrets = canAct("secrets", "create");
  const modalContentRef = useRef<HTMLFormElement>(null);

  const { data: secrets = [], isLoading } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const createSecretMutation = useCreateSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  const handleCreateClick = () => {
    if (!canCreateSecrets) return;
    setSecretName("");
    setKeyValuePairs([{ name: "", value: "" }]);
    setIsCreateModalOpen(true);
  };

  const handleCloseCreateModal = () => {
    setIsCreateModalOpen(false);
    setSecretName("");
    setKeyValuePairs([{ name: "", value: "" }]);
    createSecretMutation.reset();
  };

  const getKeyValuePairsFromForm = (): KeyValuePair[] => {
    const form = modalContentRef.current;
    if (!form) return keyValuePairs;
    const keyInputs = form.querySelectorAll<HTMLInputElement>('input[placeholder="Key"]');
    const valueInputs = form.querySelectorAll<HTMLTextAreaElement>('textarea[placeholder="Value"]');
    const pairs: KeyValuePair[] = [];
    const len = Math.min(keyInputs.length, valueInputs.length);
    for (let i = 0; i < len; i++) {
      pairs.push({ name: keyInputs[i].value ?? "", value: valueInputs[i].value ?? "" });
    }
    return pairs.length ? pairs : keyValuePairs;
  };

  const handleCreate = async () => {
    if (!canCreateSecrets) return;
    if (!secretName?.trim()) {
      showErrorToast("Secret name is required");
      return;
    }
    const formPairs = getKeyValuePairsFromForm();
    const validPairs = formPairs.filter((p) => p.name.trim() && p.value.trim());
    if (validPairs.length === 0) {
      showErrorToast("At least one key-value pair is required");
      return;
    }
    const keys = validPairs.map((p) => p.name.trim());
    if (new Set(keys).size !== keys.length) {
      showErrorToast("Duplicate key names are not allowed");
      return;
    }
    try {
      const params: CreateSecretParams = {
        name: secretName.trim(),
        environmentVariables: validPairs.map((p) => ({ name: p.name.trim(), value: p.value.trim() })),
      };
      const result = await createSecretMutation.mutateAsync(params);
      showSuccessToast("Secret created successfully");
      handleCloseCreateModal();
      const newSecretId = result?.data?.secret?.metadata?.id;
      if (newSecretId) {
        navigate(`/${organizationId}/settings/secrets/${newSecretId}`);
      }
    } catch (error) {
      showErrorToast(`Failed to create secret: ${getApiErrorMessage(error)}`);
    }
  };

  const addKeyValuePair = () => {
    setKeyValuePairs([...keyValuePairs, { name: "", value: "" }]);
  };

  const removeKeyValuePair = (index: number) => {
    setKeyValuePairs(keyValuePairs.filter((_, i) => i !== index));
  };

  const getSecretDetailPath = (id: string) => `/${organizationId}/settings/secrets/${id}`;

  if (isLoading) {
    return (
      <div className="space-y-6 pt-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
          <div className="px-6 pb-6 min-h-96 flex justify-center items-center">
            <p className="text-gray-500 dark:text-gray-400">Loading secrets...</p>
          </div>
        </div>
      </div>
    );
  }

  const sortedSecrets = [...secrets].sort((a, b) =>
    (a.metadata?.name || "").localeCompare(b.metadata?.name || ""),
  );

  return (
    <div className="space-y-6 pt-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        {sortedSecrets.length > 0 && (
          <div className="px-6 pt-6 pb-4 flex items-center justify-start">
            <PermissionTooltip
              allowed={canCreateSecrets || permissionsLoading}
              message="You don't have permission to create secrets."
            >
              <Button
                className="flex items-center"
                onClick={handleCreateClick}
                disabled={!canCreateSecrets}
              >
                <Icon name="plus" />
                Create Secret
              </Button>
            </PermissionTooltip>
          </div>
        )}
        <div className="px-6 pb-6 min-h-96">
          {sortedSecrets.length === 0 ? (
            <div className="flex min-h-96 flex-col items-center justify-center text-center">
              <div className="flex justify-center items-center text-gray-800">
                <Icon name="key" size="xl" />
              </div>
              <p className="mt-3 text-sm text-gray-800">Create your first secret</p>
              <PermissionTooltip
                allowed={canCreateSecrets || permissionsLoading}
                message="You don't have permission to create secrets."
              >
                <Button
                  className="mt-4 flex items-center"
                  onClick={handleCreateClick}
                  disabled={!canCreateSecrets}
                >
                  <Icon name="plus" />
                  Create Secret
                </Button>
              </PermissionTooltip>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Secret name</TableHeader>
                  <TableHeader>Keys</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {sortedSecrets.map((secret) => {
                  const secretId = secret.metadata?.id || "";
                  const secretData = secret.spec?.local?.data || {};
                  const keyCount = Object.keys(secretData).length;
                  return (
                    <TableRow key={secretId} className="last:[&>td]:border-b-0">
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Icon name="key" size="sm" className="text-gray-800" />
                          <Link
                            href={getSecretDetailPath(secretId)}
                            className="cursor-pointer text-sm !font-semibold text-gray-800 !underline underline-offset-2"
                          >
                            {secret.metadata?.name || "Unnamed Secret"}
                          </Link>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-gray-500 dark:text-gray-400">
                          {keyCount} key{keyCount === 1 ? "" : "s"}
                        </span>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </div>
      </div>

      {/* Create modal */}
      {isCreateModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
            <form
              ref={modalContentRef}
              className="p-6"
              onSubmit={(e) => {
                e.preventDefault();
                handleCreate();
              }}
            >
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <Key className="w-6 h-6 text-gray-500 dark:text-gray-400" />
                  <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                    Create Secret
                  </h3>
                </div>
                <button
                  type="button"
                  onClick={handleCloseCreateModal}
                  className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                  disabled={createSecretMutation.isPending}
                >
                  <Icon name="x" size="sm" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">
                    Secret Name <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    type="text"
                    value={secretName}
                    onChange={(e) => setSecretName(e.target.value)}
                    placeholder="e.g., production-api-keys"
                    required
                    disabled={!canCreateSecrets}
                  />
                </div>
                <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
                  <Label className="text-gray-800 dark:text-gray-100 mb-2 block">
                    Key-Value Pairs <span className="text-red-500">*</span>
                  </Label>
                  <div className="space-y-3">
                    {keyValuePairs.map((pair, index) => (
                      <div key={index} className="flex gap-2 items-start">
                        <div className="flex-1">
                          <Input
                            type="text"
                            defaultValue={pair.name}
                            placeholder="Key"
                            className="mb-2"
                            disabled={!canCreateSecrets}
                          />
                          <Textarea
                            defaultValue={pair.value}
                            placeholder="Value"
                            rows={3}
                            className="font-mono text-sm"
                            disabled={!canCreateSecrets}
                          />
                        </div>
                        {keyValuePairs.length > 1 && (
                          <Button
                            type="button"
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
                    ))}
                  </div>
                  <Button type="button" variant="outline" size="sm" onClick={addKeyValuePair} className="mt-3 text-xs" disabled={!canCreateSecrets}>
                    <Plus className="w-3 h-3 mr-1" />
                    Add Pair
                  </Button>
                </div>
              </div>

              <div className="flex justify-start gap-3 mt-6">
                <Button
                  type="submit"
                  color="blue"
                  disabled={createSecretMutation.isPending || !secretName?.trim() || !canCreateSecrets}
                  className="flex items-center gap-2"
                >
                  {createSecretMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    "Create Secret"
                  )}
                </Button>
                <Button type="button" variant="outline" onClick={handleCloseCreateModal} disabled={createSecretMutation.isPending}>
                  Cancel
                </Button>
              </div>

              {createSecretMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to create secret: {getApiErrorMessage(createSecretMutation.error)}
                  </p>
                </div>
              )}
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
