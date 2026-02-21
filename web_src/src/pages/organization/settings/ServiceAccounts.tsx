import { Icon } from "@/components/Icon";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Link } from "@/components/Link/link";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/Textarea/textarea";
import { usePermissions } from "@/contexts/PermissionsContext";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Bot, Copy, Loader2 } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useServiceAccounts, useCreateServiceAccount, useDeleteServiceAccount } from "@/hooks/useServiceAccounts";

interface ServiceAccountsProps {
  organizationId: string;
}

export function ServiceAccounts({ organizationId }: ServiceAccountsProps) {
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [role, setRole] = useState("org_viewer");
  const [newToken, setNewToken] = useState<string | null>(null);
  const canCreate = canAct("service_accounts", "create");
  const canDelete = canAct("service_accounts", "delete");

  const { data: serviceAccounts = [], isLoading } = useServiceAccounts(organizationId);
  const createMutation = useCreateServiceAccount(organizationId);
  const deleteMutation = useDeleteServiceAccount(organizationId);

  const handleCreateClick = () => {
    if (!canCreate) return;
    setName("");
    setDescription("");
    setRole("org_viewer");
    setNewToken(null);
    setIsCreateModalOpen(true);
  };

  const handleCloseCreateModal = () => {
    setIsCreateModalOpen(false);
    setName("");
    setDescription("");
    setRole("org_viewer");
    setNewToken(null);
    createMutation.reset();
  };

  const handleCreate = async () => {
    if (!canCreate) return;
    if (!name?.trim()) {
      showErrorToast("Name is required");
      return;
    }
    try {
      const result = await createMutation.mutateAsync({
        name: name.trim(),
        description: description.trim(),
        role,
      });
      const token = result.data?.token;
      if (token) {
        setNewToken(token);
      } else {
        showSuccessToast("Service account created");
        handleCloseCreateModal();
      }
    } catch (error) {
      showErrorToast(`Failed to create service account: ${getApiErrorMessage(error)}`);
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

  const handleTokenModalClose = () => {
    const saId = createMutation.data?.data?.serviceAccount?.id;
    handleCloseCreateModal();
    if (saId) {
      navigate(`/${organizationId}/settings/service-accounts/${saId}`);
    }
  };

  const handleDelete = async (id: string, saName: string) => {
    if (!canDelete) return;
    if (!confirm(`Are you sure you want to delete service account "${saName}"? This cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(id);
      showSuccessToast("Service account deleted");
    } catch (error) {
      showErrorToast(`Failed to delete: ${getApiErrorMessage(error)}`);
    }
  };

  const getDetailPath = (id: string) => `/${organizationId}/settings/service-accounts/${id}`;

  if (isLoading) {
    return (
      <div className="space-y-6 pt-6">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
          <div className="px-6 pb-6 min-h-96 flex justify-center items-center">
            <p className="text-gray-500 dark:text-gray-400">Loading service accounts...</p>
          </div>
        </div>
      </div>
    );
  }

  const sorted = [...serviceAccounts].sort((a, b) => (a.name || "").localeCompare(b.name || ""));

  return (
    <div className="space-y-6 pt-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        {sorted.length > 0 && (
          <div className="px-6 pt-6 pb-4 flex items-center justify-start">
            <PermissionTooltip
              allowed={canCreate || permissionsLoading}
              message="You don't have permission to create service accounts."
            >
              <Button
                className="flex items-center"
                onClick={handleCreateClick}
                disabled={!canCreate}
                data-testid="sa-create-btn"
              >
                <Icon name="plus" />
                Create Service Account
              </Button>
            </PermissionTooltip>
          </div>
        )}
        <div className="px-6 pb-6 min-h-96">
          {sorted.length === 0 ? (
            <div className="flex min-h-96 flex-col items-center justify-center text-center">
              <div className="flex justify-center items-center text-gray-800">
                <Bot size={32} />
              </div>
              <p className="mt-3 text-sm text-gray-800">Create your first service account</p>
              <p className="mt-1 text-xs text-gray-500">Service accounts provide programmatic API access.</p>
              <PermissionTooltip
                allowed={canCreate || permissionsLoading}
                message="You don't have permission to create service accounts."
              >
                <Button
                  className="mt-4 flex items-center"
                  onClick={handleCreateClick}
                  disabled={!canCreate}
                  data-testid="sa-create-btn"
                >
                  <Icon name="plus" />
                  Create Service Account
                </Button>
              </PermissionTooltip>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Name</TableHeader>
                  <TableHeader>Description</TableHeader>
                  <TableHeader>Token</TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {sorted.map((sa) => (
                  <TableRow key={sa.id} className="last:[&>td]:border-b-0">
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Bot size={16} className="text-gray-500" />
                        <Link
                          href={getDetailPath(sa.id || "")}
                          className="cursor-pointer text-sm !font-semibold text-gray-800 !underline underline-offset-2"
                          data-testid="sa-link"
                        >
                          {sa.name || "Unnamed"}
                        </Link>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-gray-500 dark:text-gray-400">{sa.description || "—"}</span>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        {sa.hasToken ? "Active" : "None"}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <PermissionTooltip
                          allowed={canDelete || permissionsLoading}
                          message="You don't have permission to delete service accounts."
                        >
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(sa.id || "", sa.name || "")}
                            disabled={!canDelete || deleteMutation.isPending}
                            className="text-red-600 hover:text-red-700"
                            data-testid="sa-delete-btn"
                          >
                            <Icon name="trash-2" size="sm" />
                          </Button>
                        </PermissionTooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </div>

      {/* Create modal */}
      {isCreateModalOpen && !newToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-lg w-full mx-4">
            <form
              className="p-6"
              onSubmit={(e) => {
                e.preventDefault();
                handleCreate();
              }}
              data-testid="sa-create-form"
            >
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <Bot className="w-6 h-6 text-gray-500 dark:text-gray-400" />
                  <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Create Service Account</h3>
                </div>
                <button
                  type="button"
                  onClick={handleCloseCreateModal}
                  className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                  disabled={createMutation.isPending}
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
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., ci-deploy-bot"
                    required
                    data-testid="sa-create-name"
                  />
                </div>
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">Description</Label>
                  <Textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder="What is this service account used for?"
                    rows={3}
                    data-testid="sa-create-description"
                  />
                </div>
                <div>
                  <Label className="text-gray-800 dark:text-gray-100 mb-2">
                    Role <span className="text-red-500">*</span>
                  </Label>
                  <Select value={role} onValueChange={setRole}>
                    <SelectTrigger className="w-full" data-testid="sa-create-role">
                      <SelectValue placeholder="Select a role" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="org_viewer">Viewer</SelectItem>
                      <SelectItem value="org_admin">Admin</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="mt-1 text-xs text-gray-500">Determines what this service account can access.</p>
                </div>
              </div>

              <div className="flex justify-start gap-3 mt-6">
                <Button
                  type="submit"
                  disabled={createMutation.isPending || !name?.trim()}
                  className="flex items-center gap-2"
                  data-testid="sa-create-submit"
                >
                  {createMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    "Create"
                  )}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleCloseCreateModal}
                  disabled={createMutation.isPending}
                >
                  Cancel
                </Button>
              </div>

              {createMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to create: {getApiErrorMessage(createMutation.error)}
                  </p>
                </div>
              )}
            </form>
          </div>
        </div>
      )}

      {/* Token display modal */}
      {isCreateModalOpen && newToken && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-lg w-full mx-4">
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <Bot className="w-6 h-6 text-green-600" />
                <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Service Account Created</h3>
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
                <Button onClick={handleTokenModalClose} data-testid="sa-token-done">
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
