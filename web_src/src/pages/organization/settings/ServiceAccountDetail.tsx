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
import {
  useServiceAccount,
  useUpdateServiceAccount,
  useDeleteServiceAccount,
  useRegenerateServiceAccountToken,
} from "@/hooks/useServiceAccounts";
import { useCanvases } from "@/hooks/useCanvasData";
import type { ServiceAccountsServiceAccount } from "@/api-client/types.gen";

interface ServiceAccountDetailProps {
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
  serviceAccount: ServiceAccountsServiceAccount | null | undefined,
  id: string | undefined,
  organizationId: string,
  canUpdate: boolean,
) {
  const updateMutation = useUpdateServiceAccount(organizationId);
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editExpiresAt, setEditExpiresAt] = useState("");
  const [editAccessMode, setEditAccessMode] = useState<AccessMode>("organization");
  const [editCanvasIds, setEditCanvasIds] = useState<string[]>([]);

  const handleEditStart = () => {
    setEditName(serviceAccount?.name || "");
    setEditDescription(serviceAccount?.description || "");
    setEditExpiresAt(toLocalDateTimeInput(serviceAccount?.expiresAt));
    setEditAccessMode(serviceAccount?.canvasIds?.length ? "canvas" : "organization");
    setEditCanvasIds(serviceAccount?.canvasIds || []);
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

    const originalExpiresAt = toLocalDateTimeInput(serviceAccount?.expiresAt);
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
  serviceAccount: ServiceAccountsServiceAccount | null | undefined,
  id: string | undefined,
  organizationId: string,
  canDelete: boolean,
  canUpdate: boolean,
) {
  const navigate = useNavigate();
  const deleteMutation = useDeleteServiceAccount(organizationId);
  const regenerateTokenMutation = useRegenerateServiceAccountToken(organizationId);
  const [newToken, setNewToken] = useState<string | null>(null);

  const handleDelete = async () => {
    if (!canDelete || !id) return;
    if (!confirm(`Are you sure you want to delete API key "${serviceAccount?.name}"? This cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(id);
      showSuccessToast("API key deleted");
      navigate(`/${organizationId}/settings/service-accounts`);
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

export function ServiceAccountDetail({ organizationId }: ServiceAccountDetailProps) {
  const { id } = useParams<{ id: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: serviceAccount, isLoading } = useServiceAccount(organizationId, id || "");
  const { data: canvases = [] } = useCanvases(organizationId);
  usePageTitle(["API Keys", serviceAccount?.name]);
  const canUpdate = canAct("service_accounts", "update");
  const canDelete = canAct("service_accounts", "delete");
  const editForm = useApiKeyEditForm(serviceAccount, id, organizationId, canUpdate);
  const tokenActions = useApiKeyTokenActions(serviceAccount, id, organizationId, canDelete, canUpdate);

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

  if (!serviceAccount) {
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

  const createdAt = serviceAccount.createdAt ? new Date(serviceAccount.createdAt).toLocaleDateString() : "—";
  const createdByLabel = serviceAccount.createdByName ? serviceAccount.createdByName.trim() : "—";
  const serviceAccountsHref = `/${organizationId}/settings/service-accounts`;
  const canvasNamesById = new Map(canvases.map((canvas) => [canvas.id, canvas.name || "Unnamed"]));
  const scopeLabel = serviceAccount.canvasIds?.length
    ? serviceAccount.canvasIds.map((canvasId) => canvasNamesById.get(canvasId) || canvasId).join(", ")
    : "Organization-wide";

  return (
    <div className="space-y-6 pt-6">
      {/* Back button */}
      <Link
        to={serviceAccountsHref}
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
              <h2 className="text-lg font-semibold text-gray-800 dark:text-white">{serviceAccount.name}</h2>
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
                    data-testid="sa-detail-edit"
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
                  data-testid="sa-detail-delete"
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
                  data-testid="sa-detail-edit-name"
                />
              </div>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Description</Label>
                <Textarea
                  value={editForm.editDescription}
                  onChange={(e) => editForm.setEditDescription(e.target.value)}
                  rows={3}
                  data-testid="sa-detail-edit-description"
                />
              </div>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Access</Label>
                <Select
                  value={editForm.editAccessMode}
                  onValueChange={(value) => editForm.setEditAccessMode(value as AccessMode)}
                >
                  <SelectTrigger className="w-full" data-testid="sa-detail-edit-access-mode">
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
                            data-testid="sa-detail-edit-canvas"
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
                  data-testid="sa-detail-edit-expires-at"
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
              <dd className="text-gray-800 dark:text-white">{serviceAccount.description || "—"}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Access</dt>
              <dd className="text-gray-800 dark:text-white">{scopeLabel}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Expires</dt>
              <dd className="text-gray-800 dark:text-white">{formatDateTime(serviceAccount.expiresAt)}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Created by</dt>
              <dd className="text-gray-800 dark:text-white">{createdByLabel}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Created at</dt>
              <dd className="text-gray-800 dark:text-white">{createdAt}</dd>
              <dt className="text-gray-500 dark:text-gray-400">ID</dt>
              <dd className="text-gray-800 dark:text-white font-mono text-xs">{serviceAccount.id}</dd>
            </dl>
          )}
        </div>
      </div>

      {/* Token management */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 py-6">
          <h3 className="text-sm font-semibold text-gray-800 dark:text-white mb-2">API Token</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
            {serviceAccount.hasToken
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
              data-testid="sa-detail-regenerate-token"
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
                  data-testid="sa-token-display"
                />
                <CopyButton
                  variant="button"
                  text={tokenActions.newToken}
                  data-testid="sa-token-copy"
                  onCopyError={() => showErrorToast("Failed to copy token")}
                >
                  Copy
                </CopyButton>
              </div>

              <div className="flex justify-start mt-6">
                <Button onClick={() => tokenActions.setNewToken(null)} data-testid="sa-token-done">
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
