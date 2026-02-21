import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/Textarea/textarea";
import { usePermissions } from "@/contexts/PermissionsContext";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { Bot, Copy, Loader2, ArrowLeft } from "lucide-react";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  useServiceAccount,
  useUpdateServiceAccount,
  useDeleteServiceAccount,
  useRegenerateServiceAccountToken,
} from "@/hooks/useServiceAccounts";

interface ServiceAccountDetailProps {
  organizationId: string;
}

export function ServiceAccountDetail({ organizationId }: ServiceAccountDetailProps) {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("service_accounts", "update");
  const canDelete = canAct("service_accounts", "delete");

  const { data: serviceAccount, isLoading } = useServiceAccount(organizationId, id || "");
  const updateMutation = useUpdateServiceAccount(organizationId);
  const deleteMutation = useDeleteServiceAccount(organizationId);
  const regenerateTokenMutation = useRegenerateServiceAccountToken(organizationId);

  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [newToken, setNewToken] = useState<string | null>(null);

  const handleEditStart = () => {
    setEditName(serviceAccount?.name || "");
    setEditDescription(serviceAccount?.description || "");
    setIsEditing(true);
  };

  const handleEditCancel = () => {
    setIsEditing(false);
  };

  const handleEditSave = async () => {
    if (!canUpdate || !id) return;
    if (!editName?.trim()) {
      showErrorToast("Name is required");
      return;
    }
    try {
      await updateMutation.mutateAsync({
        id,
        name: editName.trim(),
        description: editDescription.trim(),
      });
      showSuccessToast("Service account updated");
      setIsEditing(false);
    } catch (error) {
      showErrorToast(`Failed to update: ${getApiErrorMessage(error)}`);
    }
  };

  const handleDelete = async () => {
    if (!canDelete || !id) return;
    if (!confirm(`Are you sure you want to delete "${serviceAccount?.name}"? This cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(id);
      showSuccessToast("Service account deleted");
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

  const handleCopyToken = async () => {
    if (!newToken) return;
    try {
      await navigator.clipboard.writeText(newToken);
      showSuccessToast("Token copied to clipboard");
    } catch {
      showErrorToast("Failed to copy token");
    }
  };

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
            <p className="text-gray-500 dark:text-gray-400">Service account not found</p>
          </div>
        </div>
      </div>
    );
  }

  const createdAt = serviceAccount.createdAt ? new Date(serviceAccount.createdAt).toLocaleDateString() : "—";

  return (
    <div className="space-y-6 pt-6">
      {/* Back button */}
      <button
        type="button"
        onClick={() => navigate(`/${organizationId}/settings/service-accounts`)}
        className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-800 transition"
      >
        <ArrowLeft size={14} />
        Back to service accounts
      </button>

      {/* Details */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 py-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <Bot size={20} className="text-gray-500" />
              <h2 className="text-lg font-semibold text-gray-800 dark:text-white">{serviceAccount.name}</h2>
            </div>
            <div className="flex gap-2">
              {!isEditing && (
                <PermissionTooltip
                  allowed={canUpdate || permissionsLoading}
                  message="You don't have permission to update service accounts."
                >
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleEditStart}
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
                message="You don't have permission to delete service accounts."
              >
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleDelete}
                  disabled={!canDelete || deleteMutation.isPending}
                  className="text-red-600 hover:text-red-700"
                  data-testid="sa-detail-delete"
                >
                  <Icon name="trash-2" size="sm" />
                  Delete
                </Button>
              </PermissionTooltip>
            </div>
          </div>

          {isEditing ? (
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault();
                handleEditSave();
              }}
            >
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">
                  Name <span className="text-red-500">*</span>
                </Label>
                <Input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  required
                  data-testid="sa-detail-edit-name"
                />
              </div>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Description</Label>
                <Textarea
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                  rows={3}
                  data-testid="sa-detail-edit-description"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  disabled={updateMutation.isPending || !editName?.trim()}
                  className="flex items-center gap-2"
                >
                  {updateMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    "Save"
                  )}
                </Button>
                <Button type="button" variant="outline" onClick={handleEditCancel} disabled={updateMutation.isPending}>
                  Cancel
                </Button>
              </div>
            </form>
          ) : (
            <dl className="grid grid-cols-2 gap-y-4 text-sm">
              <dt className="text-gray-500 dark:text-gray-400">Description</dt>
              <dd className="text-gray-800 dark:text-white">{serviceAccount.description || "—"}</dd>
              <dt className="text-gray-500 dark:text-gray-400">Created</dt>
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
              ? "This service account has an active token. Regenerating will invalidate the current one."
              : "No token is currently active for this service account."}
          </p>
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to manage service account tokens."
          >
            <Button
              variant="outline"
              onClick={handleRegenerateToken}
              disabled={!canUpdate || regenerateTokenMutation.isPending}
              data-testid="sa-detail-regenerate-token"
            >
              {regenerateTokenMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin mr-1" />
                  Regenerating...
                </>
              ) : (
                <>
                  <Icon name="refresh-cw" size="sm" />
                  Regenerate Token
                </>
              )}
            </Button>
          </PermissionTooltip>
        </div>
      </div>

      {/* Token display modal */}
      {newToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-lg w-full mx-4">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <Bot className="w-6 h-6 text-green-600" />
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
                  value={newToken}
                  className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-800"
                  data-testid="sa-token-display"
                />
                <Button variant="outline" onClick={handleCopyToken} data-testid="sa-token-copy">
                  <Copy className="w-4 h-4" />
                </Button>
              </div>

              <div className="flex justify-start mt-6">
                <Button onClick={() => setNewToken(null)} data-testid="sa-token-done">
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
