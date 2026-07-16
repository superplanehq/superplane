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
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ArrowLeft, KeyRound } from "lucide-react";
import { CopyButton } from "@/ui/CopyButton";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAPIKey, useUpdateAPIKey, useDeleteAPIKey, useRegenerateAPIKeyToken } from "@/hooks/useApiKeys";
import { useCanvases } from "@/hooks/useCanvasData";
import type { ApiKeysApiKey } from "@/api-client/types.gen";

interface APIKeyDetailProps {
  organizationId: string;
}

type AccessMode = "organization" | "canvas";

function toApiTimestamp(localValue: string) {
  if (!localValue) return undefined;
  return new Date(localValue).toISOString();
}

function toLocalDateTimeInput(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  const offsetMs = date.getTimezoneOffset() * 60 * 1000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}

function formatDateTime(value?: string) {
  if (!value) return "Never";
  return new Date(value).toLocaleString();
}

function useApiKeyEditForm(
  apiKey: ApiKeysApiKey | null | undefined,
  id: string | undefined,
  organizationId: string,
  canUpdate: boolean,
) {
  const updateMutation = useUpdateAPIKey(organizationId);
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editExpiresAt, setEditExpiresAt] = useState("");
  const [editAccessMode, setEditAccessMode] = useState<AccessMode>("organization");
  const [editCanvasIds, setEditCanvasIds] = useState<string[]>([]);

  const handleEditStart = () => {
    setEditName(apiKey?.name || "");
    setEditDescription(apiKey?.description || "");
    setEditExpiresAt(toLocalDateTimeInput(apiKey?.expiresAt));
    setEditAccessMode(apiKey?.canvasIds?.length ? "canvas" : "organization");
    setEditCanvasIds(apiKey?.canvasIds || []);
    setIsEditing(true);
  };

  const handleEditCancel = () => {
    setIsEditing(false);
  };

  const toggleEditCanvas = (canvasId: string) => {
    setEditCanvasIds((current) =>
      current.includes(canvasId) ? current.filter((selectedId) => selectedId !== canvasId) : [...current, canvasId],
    );
  };

  const handleEditSave = async () => {
    if (!canUpdate || !id) return;
    if (!editName?.trim()) {
      showErrorToast("Name is required");
      return;
    }
    if (editAccessMode === "canvas" && editCanvasIds.length === 0) {
      showErrorToast("Select at least one app");
      return;
    }

    const originalExpiresAt = toLocalDateTimeInput(apiKey?.expiresAt);
    const expiresAtChanged = editExpiresAt !== originalExpiresAt;

    try {
      await updateMutation.mutateAsync({
        id,
        name: editName.trim(),
        description: editDescription.trim(),
        expiresAt: expiresAtChanged ? toApiTimestamp(editExpiresAt) : undefined,
        clearExpiresAt: expiresAtChanged && !editExpiresAt,
        canvasIds: editAccessMode === "canvas" ? editCanvasIds : [],
      });
      showSuccessToast("API key updated");
      setIsEditing(false);
    } catch (error) {
      showErrorToast(`Failed to update: ${getApiErrorMessage(error)}`);
    }
  };

  return {
    isEditing,
    editName,
    setEditName,
    editDescription,
    setEditDescription,
    editExpiresAt,
    setEditExpiresAt,
    editAccessMode,
    setEditAccessMode,
    editCanvasIds,
    updateMutation,
    handleEditStart,
    handleEditCancel,
    handleEditSave,
    toggleEditCanvas,
  };
}

function useApiKeyTokenActions(
  apiKey: ApiKeysApiKey | null | undefined,
  id: string | undefined,
  organizationId: string,
  canDelete: boolean,
  canUpdate: boolean,
) {
  const navigate = useNavigate();
  const deleteMutation = useDeleteAPIKey(organizationId);
  const regenerateTokenMutation = useRegenerateAPIKeyToken(organizationId);
  const [newToken, setNewToken] = useState<string | null>(null);

  const handleDelete = async () => {
    if (!canDelete || !id) return;
    if (!confirm(`Are you sure you want to delete API key "${apiKey?.name}"? This cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(id);
      showSuccessToast("API key deleted");
      navigate(`/${organizationId}/settings/api-keys`);
    } catch (error) {
      showErrorToast(`Failed to delete: ${getApiErrorMessage(error)}`);
    }
  };

  const handleRegenerateToken = async () => {
    if (!canUpdate || !id) return;
    if (!confirm("Are you sure? The current token will stop working immediately.")) return;
    try {
      const result = await regenerateTokenMutation.mutateAsync(id);
      const token = result.data?.token;
      if (token) {
        setNewToken(token);
      }
    } catch (error) {
      showErrorToast(`Failed to regenerate token: ${getApiErrorMessage(error)}`);
    }
  };

  return {
    deleteMutation,
    regenerateTokenMutation,
    newToken,
    setNewToken,
    handleDelete,
    handleRegenerateToken,
  };
}

export function APIKeyDetail({ organizationId }: APIKeyDetailProps) {
  const { id } = useParams<{ id: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: apiKey, isLoading } = useAPIKey(organizationId, id || "");
  const { data: canvases = [] } = useCanvases(organizationId);
  usePageTitle(["API Keys", apiKey?.name]);
  const canUpdate = canAct("api_keys", "update");
  const canDelete = canAct("api_keys", "delete");
  const editForm = useApiKeyEditForm(apiKey, id, organizationId, canUpdate);
  const tokenActions = useApiKeyTokenActions(apiKey, id, organizationId, canDelete, canUpdate);

  useReportPageReady(!isLoading && !permissionsLoading && !!id);

  if (isLoading) {
    return (
      <div className="space-y-6 pt-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
          <div className="px-6 pb-6 min-h-96 flex justify-center items-center">
            <p className="text-gray-500 dark:text-gray-400">Loading...</p>
          </div>
        </div>
      </div>
    );
  }

  if (!apiKey) {
    return (
      <div className="space-y-6 pt-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
          <div className="px-6 pb-6 min-h-96 flex justify-center items-center">
            <p className="text-gray-500 dark:text-gray-400">API key not found</p>
          </div>
        </div>
      </div>
    );
  }

  const createdAt = apiKey.createdAt ? new Date(apiKey.createdAt).toLocaleDateString() : "—";
  const createdByLabel = apiKey.createdByName ? apiKey.createdByName.trim() : "—";
  const apiKeysHref = `/${organizationId}/settings/api-keys`;
  const canvasNamesById = new Map(canvases.map((canvas) => [canvas.id, canvas.name || "Unnamed"]));
  const scopeLabel = apiKey.canvasIds?.length
    ? apiKey.canvasIds.map((canvasId) => canvasNamesById.get(canvasId) || canvasId).join(", ")
    : "Organization-wide";

  return (
    <div className="space-y-6 pt-6">
      {/* Back button */}
      <Link
        to={apiKeysHref}
        className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-800 transition"
        aria-label="Back to API keys"
      >
        <ArrowLeft size={14} />
        Back to API keys
      </Link>

      {/* Details */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 py-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <KeyRound size={20} className="text-gray-500" />
              <h2 className="text-lg font-semibold text-gray-800 dark:text-white">{apiKey.name}</h2>
            </div>
            <div className="flex gap-2">
              {!editForm.isEditing && (
                <PermissionTooltip
                  allowed={canUpdate || permissionsLoading}
                  message="You don't have permission to update API keys."
                >
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={editForm.handleEditStart}
                    disabled={!canUpdate}
                    data-testid="api-key-detail-edit"
                  >
                    <Icon name="pencil" size="sm" />
                    Edit
                  </Button>
                </PermissionTooltip>
              )}
              <PermissionTooltip
                allowed={canDelete || permissionsLoading}
                message="You don't have permission to delete API keys."
              >
                <Button
                  variant="outline"
                  size="sm"
                  onClick={tokenActions.handleDelete}
                  disabled={!canDelete || tokenActions.deleteMutation.isPending}
                  className="text-red-600 hover:text-red-700"
                  data-testid="api-key-detail-delete"
                >
                  <Icon name="trash-2" size="sm" />
                  Delete
                </Button>
              </PermissionTooltip>
            </div>
          </div>

          {editForm.isEditing ? (
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault();
                editForm.handleEditSave();
              }}
            >
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">
                  Name <span className="text-red-500">*</span>
                </Label>
                <Input
                  type="text"
                  value={editForm.editName}
                  onChange={(e) => editForm.setEditName(e.target.value)}
                  required
                  data-testid="api-key-detail-edit-name"
                />
              </div>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Description</Label>
                <Textarea
                  value={editForm.editDescription}
                  onChange={(e) => editForm.setEditDescription(e.target.value)}
                  rows={3}
                  data-testid="api-key-detail-edit-description"
                />
              </div>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Access</Label>
                <Select
                  value={editForm.editAccessMode}
                  onValueChange={(value) => editForm.setEditAccessMode(value as AccessMode)}
                >
                  <SelectTrigger className="w-full" data-testid="api-key-detail-edit-access-mode">
                    <SelectValue placeholder="Select access" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="organization">Organization-wide</SelectItem>
                    <SelectItem value="canvas">Selected apps</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {editForm.editAccessMode === "canvas" && (
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
                            checked={editForm.editCanvasIds.includes(canvasId)}
                            onChange={() => editForm.toggleEditCanvas(canvasId)}
                            data-testid="api-key-detail-edit-canvas"
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
                  value={editForm.editExpiresAt}
                  onChange={(e) => editForm.setEditExpiresAt(e.target.value)}
                  data-testid="api-key-detail-edit-expires-at"
                />
              </div>
              <div className="flex gap-2">
                <LoadingButton
                  type="submit"
                  disabled={!editForm.editName?.trim()}
                  loading={editForm.updateMutation.isPending}
                  loadingText="Saving..."
                  className="flex items-center gap-2"
                >
                  Save
                </LoadingButton>
                <Button
                  type="button"
                  variant="outline"
                  onClick={editForm.handleEditCancel}
                  disabled={editForm.updateMutation.isPending}
                >
                  Cancel
                </Button>
              </div>
            </form>
          ) : (
            <dl className="grid grid-cols-2 gap-y-4 text-sm">
              <dt className="text-gray-500 dark:text-gray-400">Description</dt>
              <dd className="text-gray-800 dark:text-white">{apiKey.description || "—"}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Access</dt>
              <dd className="text-gray-800 dark:text-white">{scopeLabel}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Expires</dt>
              <dd className="text-gray-800 dark:text-white">{formatDateTime(apiKey.expiresAt)}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Created by</dt>
              <dd className="text-gray-800 dark:text-white">{createdByLabel}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Created at</dt>
              <dd className="text-gray-800 dark:text-white">{createdAt}</dd>
              <dt className="text-gray-500 dark:text-gray-400">ID</dt>
              <dd className="text-gray-800 dark:text-white font-mono text-xs">{apiKey.id}</dd>
            </dl>
          )}
        </div>
      </div>

      {/* Token management */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 py-6">
          <h3 className="text-sm font-semibold text-gray-800 dark:text-white mb-2">API Token</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
            {apiKey.hasToken
              ? "This API key has an active token. Regenerating will invalidate the current one."
              : "No token is currently active for this API key."}
          </p>
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to manage API key tokens."
          >
            <LoadingButton
              variant="outline"
              onClick={tokenActions.handleRegenerateToken}
              disabled={!canUpdate}
              loading={tokenActions.regenerateTokenMutation.isPending}
              loadingText="Regenerating..."
              data-testid="api-key-detail-regenerate-token"
            >
              <Icon name="refresh-cw" size="sm" />
              Regenerate Token
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </div>

      {/* Token display modal */}
      {tokenActions.newToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-lg w-full mx-4">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <KeyRound className="w-6 h-6 text-green-600" />
                <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Token Regenerated</h3>
              </div>

              <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md mb-4">
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  Copy this token now. You won't be able to see it again.
                </p>
              </div>

              <div className="flex items-center gap-2">
                <Input
                  readOnly
                  value={tokenActions.newToken}
                  className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-800"
                  data-testid="api-key-token-display"
                />
                <CopyButton
                  variant="button"
                  text={tokenActions.newToken}
                  data-testid="api-key-token-copy"
                  onCopyError={() => showErrorToast("Failed to copy token")}
                >
                  Copy
                </CopyButton>
              </div>

              <div className="flex justify-start mt-6">
                <Button onClick={() => tokenActions.setNewToken(null)} data-testid="api-key-token-done">
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
