import { Icon } from "@/components/Icon";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/Textarea/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { settingsModalClassName, settingsTableCardClassName } from "./settingsPageStyles";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { KeyRound } from "lucide-react";
import { CopyButton } from "@/ui/CopyButton";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAPIKeys, useCreateAPIKey, useDeleteAPIKey } from "@/hooks/useApiKeys";
import { useCanvases } from "@/hooks/useCanvasData";
import { useOrganizationRoles } from "@/hooks/useOrganizationData";
import { ApiKeysContent } from "./ApiKeysContent";

interface APIKeysProps {
  organizationId: string;
}

type AccessMode = "organization" | "canvas";

function toApiTimestamp(localValue: string) {
  if (!localValue) return undefined;
  return new Date(localValue).toISOString();
}

function useCreateApiKeyForm(organizationId: string, canCreate: boolean) {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [role, setRole] = useState("org_viewer");
  const [expiresAt, setExpiresAt] = useState("");
  const [accessMode, setAccessMode] = useState<AccessMode>("organization");
  const [selectedCanvasIds, setSelectedCanvasIds] = useState<string[]>([]);
  const [newToken, setNewToken] = useState<string | null>(null);
  const navigate = useNavigate();
  const createMutation = useCreateAPIKey(organizationId);

  const handleCreateClick = () => {
    if (!canCreate) return;
    setName("");
    setDescription("");
    setRole("org_viewer");
    setExpiresAt("");
    setAccessMode("organization");
    setSelectedCanvasIds([]);
    setNewToken(null);
    setIsCreateModalOpen(true);
  };

  const handleCloseCreateModal = () => {
    setIsCreateModalOpen(false);
    setName("");
    setDescription("");
    setRole("org_viewer");
    setExpiresAt("");
    setAccessMode("organization");
    setSelectedCanvasIds([]);
    setNewToken(null);
    createMutation.reset();
  };

  const handleCreate = async () => {
    if (!canCreate) return;
    if (!name?.trim()) {
      showErrorToast("Name is required");
      return;
    }
    if (accessMode === "canvas" && selectedCanvasIds.length === 0) {
      showErrorToast("Select at least one app");
      return;
    }

    try {
      const result = await createMutation.mutateAsync({
        name: name.trim(),
        description: description.trim(),
        role,
        expiresAt: toApiTimestamp(expiresAt),
        canvasIds: accessMode === "canvas" ? selectedCanvasIds : [],
      });
      const token = result.data?.token;
      if (token) {
        setNewToken(token);
      } else {
        showSuccessToast("API key created");
        handleCloseCreateModal();
      }
    } catch (error) {
      showErrorToast(`Failed to create API key: ${getApiErrorMessage(error)}`);
    }
  };

  const handleTokenModalClose = () => {
    const apiKeyId = createMutation.data?.data?.apiKey?.id;
    handleCloseCreateModal();
    if (apiKeyId) {
      navigate(`/${organizationId}/settings/api-keys/${apiKeyId}`);
    }
  };

  const toggleCanvas = (canvasId: string) => {
    setSelectedCanvasIds((current) =>
      current.includes(canvasId) ? current.filter((id) => id !== canvasId) : [...current, canvasId],
    );
  };

  return {
    isCreateModalOpen,
    name,
    setName,
    description,
    setDescription,
    role,
    setRole,
    expiresAt,
    setExpiresAt,
    accessMode,
    setAccessMode,
    selectedCanvasIds,
    newToken,
    createMutation,
    handleCreateClick,
    handleCloseCreateModal,
    handleCreate,
    handleTokenModalClose,
    toggleCanvas,
  };
}

export function APIKeys({ organizationId }: APIKeysProps) {
  usePageTitle(["API Keys"]);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canCreate = canAct("api_keys", "create");
  const canDelete = canAct("api_keys", "delete");

  const { data: apiKeys = [], isLoading } = useAPIKeys(organizationId);
  const { data: canvases = [] } = useCanvases(organizationId);
  const { data: roles = [] } = useOrganizationRoles(organizationId);
  const deleteMutation = useDeleteAPIKey(organizationId);
  const form = useCreateApiKeyForm(organizationId, canCreate);

  // org_owner is reserved for human users and cannot be assigned to an API key.
  const assignableRoles = useMemo(() => {
    const builtinNames = new Set(["org_admin", "org_viewer"]);
    const isBuiltin = (name: string) => builtinNames.has(name);
    const customRoles = roles
      .filter((role) => {
        const name = role.metadata?.name || "";
        return name !== "org_owner" && !isBuiltin(name);
      })
      .sort((a, b) => (a.spec?.displayName || "").localeCompare(b.spec?.displayName || ""));
    const baseRoles = roles
      .filter((role) => isBuiltin(role.metadata?.name || ""))
      .sort((a, b) => (a.spec?.displayName || "").localeCompare(b.spec?.displayName || ""));
    return [...customRoles, ...baseRoles];
  }, [roles]);

  useReportPageReady(!isLoading && !permissionsLoading);

  const handleDelete = async (id: string, apiKeyName: string) => {
    if (!canDelete) return;
    if (!confirm(`Are you sure you want to delete API key "${apiKeyName}"? This cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(id);
      showSuccessToast("API key deleted");
    } catch (error) {
      showErrorToast(`Failed to delete: ${getApiErrorMessage(error)}`);
    }
  };

  const getDetailPath = (id: string) => `/${organizationId}/settings/api-keys/${id}`;
  const canvasNamesById = new Map(canvases.map((canvas) => [canvas.id, canvas.name || "Unnamed"]));
  const scopeLabel = (canvasIds?: string[]) => {
    if (!canvasIds || canvasIds.length === 0) return "Organization-wide";
    if (canvasIds.length === 1) return canvasNamesById.get(canvasIds[0]) || "1 selected app";
    return `${canvasIds.length} selected apps`;
  };

  if (isLoading) {
    return (
      <div className="space-y-6 pt-6">
        <div className={settingsTableCardClassName}>
          <div className="flex min-h-96 items-center justify-center px-6 pb-6">
            <p className="text-gray-500 dark:text-gray-400">Loading API keys...</p>
          </div>
        </div>
      </div>
    );
  }

  const sorted = [...apiKeys].sort((a, b) => (a.name || "").localeCompare(b.name || ""));

  return (
    <div className="space-y-6 pt-6">
      <div className={settingsTableCardClassName}>
        {sorted.length > 0 && (
          <div className="px-6 pt-6 pb-4 flex items-center justify-start">
            <PermissionTooltip
              allowed={canCreate || permissionsLoading}
              message="You don't have permission to create API keys."
            >
              <Button
                className="flex items-center"
                onClick={form.handleCreateClick}
                disabled={!canCreate}
                data-testid="api-key-create-btn"
              >
                <Icon name="plus" />
                Create API Key
              </Button>
            </PermissionTooltip>
          </div>
        )}
        <div className="px-6 pb-6 min-h-96">
          <ApiKeysContent
            sorted={sorted}
            canCreate={canCreate}
            canDelete={canDelete}
            permissionsLoading={permissionsLoading}
            deletePending={deleteMutation.isPending}
            onCreateClick={form.handleCreateClick}
            onDelete={handleDelete}
            getDetailPath={getDetailPath}
            scopeLabel={scopeLabel}
          />
        </div>
      </div>
      {form.isCreateModalOpen && !form.newToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className={cn(settingsModalClassName, "max-w-lg")}>
            <form
              className="p-6"
              onSubmit={(e) => {
                e.preventDefault();
                form.handleCreate();
              }}
              data-testid="api-key-create-form"
            >
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <KeyRound className="w-6 h-6 text-gray-500 dark:text-gray-400" />
                  <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Create API Key</h3>
                </div>
                <button
                  type="button"
                  onClick={form.handleCloseCreateModal}
                  className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                  disabled={form.createMutation.isPending}
                >
                  <Icon name="x" size="sm" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">
                    Name <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    type="text"
                    value={form.name}
                    onChange={(e) => form.setName(e.target.value)}
                    placeholder="e.g., ci-deploy-bot"
                    required
                    data-testid="api-key-create-name"
                  />
                </div>
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">Description</Label>
                  <Textarea
                    value={form.description}
                    onChange={(e) => form.setDescription(e.target.value)}
                    placeholder="What is this API key used for?"
                    rows={3}
                    data-testid="api-key-create-description"
                  />
                </div>
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">
                    Role <span className="text-red-500">*</span>
                  </Label>
                  <Select value={form.role} onValueChange={form.setRole}>
                    <SelectTrigger className="w-full" data-testid="api-key-create-role">
                      <SelectValue placeholder="Select a role" />
                    </SelectTrigger>
                    <SelectContent>
                      {assignableRoles.map((role) => (
                        <SelectItem key={role.metadata?.name} value={role.metadata?.name || ""}>
                          {role.spec?.displayName || role.metadata?.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Determines what this API key can do within its access scope.
                  </p>
                </div>
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">Access</Label>
                  <Select value={form.accessMode} onValueChange={(value) => form.setAccessMode(value as AccessMode)}>
                    <SelectTrigger className="w-full" data-testid="api-key-create-access-mode">
                      <SelectValue placeholder="Select access" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="organization">Organization-wide</SelectItem>
                      <SelectItem value="canvas">Selected apps</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                {form.accessMode === "canvas" && (
                  <div>
                    <Label className="text-gray-800 dark:text-gray-100 mb-2">
                      Apps <span className="text-red-500">*</span>
                    </Label>
                    <div className="max-h-44 overflow-y-auto rounded-md border border-gray-200 dark:border-gray-700">
                      {canvases.map((canvas) => {
                        const canvasId = canvas.id || "";
                        return (
                          <label
                            key={canvasId}
                            className="flex items-center gap-2 border-b border-gray-100 px-3 py-2 text-sm last:border-b-0 dark:border-gray-800"
                          >
                            <Checkbox
                              checked={form.selectedCanvasIds.includes(canvasId)}
                              onChange={() => form.toggleCanvas(canvasId)}
                              data-testid="api-key-create-canvas"
                            />
                            <span className="text-gray-800 dark:text-gray-100">{canvas.name || "Unnamed"}</span>
                          </label>
                        );
                      })}
                    </div>
                  </div>
                )}
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">Expiration</Label>
                  <Input
                    type="datetime-local"
                    value={form.expiresAt}
                    onChange={(e) => form.setExpiresAt(e.target.value)}
                    data-testid="api-key-create-expires-at"
                  />
                </div>
              </div>

              <div className="flex justify-start gap-3 mt-6">
                <LoadingButton
                  type="submit"
                  disabled={!form.name?.trim()}
                  loading={form.createMutation.isPending}
                  loadingText="Creating..."
                  className="flex items-center gap-2"
                  data-testid="api-key-create-submit"
                >
                  Create
                </LoadingButton>
                <Button
                  type="button"
                  variant="outline"
                  onClick={form.handleCloseCreateModal}
                  disabled={form.createMutation.isPending}
                >
                  Cancel
                </Button>
              </div>

              {form.createMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to create: {getApiErrorMessage(form.createMutation.error)}
                  </p>
                </div>
              )}
            </form>
          </div>
        </div>
      )}
      {form.isCreateModalOpen && form.newToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className={cn(settingsModalClassName, "max-w-lg")}>
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <KeyRound className="h-6 w-6 text-green-600 dark:text-green-400" />
                <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">API Key Created</h3>
              </div>

              <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md mb-4">
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  Copy this token now. You won't be able to see it again.
                </p>
              </div>

              <div className="flex items-center gap-2 ph-no-capture">
                <Input
                  readOnly
                  value={form.newToken}
                  className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-800"
                  data-testid="api-key-token-display"
                />
                <CopyButton
                  variant="button"
                  text={form.newToken}
                  data-testid="api-key-token-copy"
                  onCopyError={() => showErrorToast("Failed to copy token")}
                >
                  Copy
                </CopyButton>
              </div>

              <div className="flex justify-start mt-6">
                <Button onClick={form.handleTokenModalClose} data-testid="api-key-token-done">
                  Done
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
