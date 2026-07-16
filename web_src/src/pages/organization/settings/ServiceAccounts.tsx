import { Icon } from "@/components/Icon";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Link } from "@/components/Link/link";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/Table/table";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/Textarea/textarea";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  settingsEmptyStateIconClassName,
  settingsEmptyStateSubtitleClassName,
  settingsEmptyStateTitleClassName,
  settingsModalClassName,
  settingsTableCardClassName,
  settingsTableLinkClassName,
} from "./settingsPageStyles";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Bot } from "lucide-react";
import { CopyButton } from "@/ui/CopyButton";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useServiceAccounts, useCreateServiceAccount, useDeleteServiceAccount } from "@/hooks/useServiceAccounts";
import { useOrganizationRoles } from "@/hooks/useOrganizationData";

interface ServiceAccountsProps {
  organizationId: string;
}

export function ServiceAccounts({ organizationId }: ServiceAccountsProps) {
  usePageTitle(["Service Accounts"]);
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
  const { data: roles = [], isLoading: rolesLoading } = useOrganizationRoles(organizationId);
  const createMutation = useCreateServiceAccount(organizationId);
  const deleteMutation = useDeleteServiceAccount(organizationId);

  // List custom roles first, then default roles, each sorted alphabetically —
  // matching the role dropdown UX used when creating groups.
  const sortedRoles = useMemo(() => {
    const defaultRoles = new Set(["org_admin", "org_owner", "org_viewer"]);
    const byDisplayName = (a: (typeof roles)[number], b: (typeof roles)[number]) =>
      (a.spec?.displayName || a.metadata?.name || "").localeCompare(b.spec?.displayName || b.metadata?.name || "");
    const customRoles = roles.filter((role) => !defaultRoles.has(role.metadata?.name || "")).sort(byDisplayName);
    const baseRoles = roles.filter((role) => defaultRoles.has(role.metadata?.name || "")).sort(byDisplayName);
    return [...customRoles, ...baseRoles];
  }, [roles]);

  const selectedRoleDescription = sortedRoles.find((r) => r.metadata?.name === role)?.spec?.description;

  useReportPageReady(!isLoading && !permissionsLoading);

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
        <div className={settingsTableCardClassName}>
          <div className="flex min-h-96 items-center justify-center px-6 pb-6">
            <p className="text-gray-500 dark:text-gray-400">Loading service accounts...</p>
          </div>
        </div>
      </div>
    );
  }

  const sorted = [...serviceAccounts].sort((a, b) => (a.name || "").localeCompare(b.name || ""));

  return (
    <div className="space-y-6 pt-6">
      <div className={settingsTableCardClassName}>
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
              <div className={cn("flex items-center justify-center", settingsEmptyStateIconClassName)}>
                <Bot size={32} />
              </div>
              <p className={settingsEmptyStateTitleClassName}>Create your first service account</p>
              <p className={settingsEmptyStateSubtitleClassName}>Service accounts provide programmatic API access.</p>
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
                  <TableHeader>Created by</TableHeader>
                  <TableHeader>Token</TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {sorted.map((sa) => (
                  <TableRow key={sa.id} className="last:[&>td]:border-b-0">
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Bot size={16} className="text-gray-500 dark:text-gray-400" />
                        <Link
                          href={getDetailPath(sa.id || "")}
                          className={settingsTableLinkClassName}
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
                        {sa.createdByName ? sa.createdByName?.trim() : "—"}
                      </span>
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
                            className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
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
          <div className={cn(settingsModalClassName, "max-w-lg")}>
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
                  <Select value={role} onValueChange={setRole} disabled={rolesLoading}>
                    <SelectTrigger className="w-full" data-testid="sa-create-role">
                      <SelectValue placeholder={rolesLoading ? "Loading roles..." : "Select a role"} />
                    </SelectTrigger>
                    <SelectContent>
                      {sortedRoles.map((r) => (
                        <SelectItem key={r.metadata?.name} value={r.metadata?.name || ""}>
                          {r.spec?.displayName || r.metadata?.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {selectedRoleDescription || "Determines what this service account can access."}
                  </p>
                </div>
              </div>

              <div className="flex justify-start gap-3 mt-6">
                <LoadingButton
                  type="submit"
                  disabled={!name?.trim()}
                  loading={createMutation.isPending}
                  loadingText="Creating..."
                  className="flex items-center gap-2"
                  data-testid="sa-create-submit"
                >
                  Create
                </LoadingButton>
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
          <div className={cn(settingsModalClassName, "max-w-lg")}>
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <Bot className="h-6 w-6 text-green-600 dark:text-green-400" />
                <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Service Account Created</h3>
              </div>

              <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md mb-4">
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  Copy this token now. You won't be able to see it again.
                </p>
              </div>

              <div className="flex items-center gap-2 ph-no-capture">
                <Input
                  readOnly
                  value={newToken}
                  className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-800"
                  data-testid="sa-token-display"
                />
                <CopyButton
                  variant="button"
                  text={newToken}
                  data-testid="sa-token-copy"
                  onCopyError={() => showErrorToast("Failed to copy token")}
                >
                  Copy
                </CopyButton>
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
