import { Icon } from "@/components/Icon";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { settingsTableCardClassName } from "./settingsPageStyles";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAPIKeys, useCreateAPIKey, useDeleteAPIKey } from "@/hooks/useApiKeys";
import { useCanvases } from "@/hooks/useCanvasData";
import { useOrganizationRoles } from "@/hooks/useOrganizationData";
import { ApiKeysContent } from "./ApiKeysContent";
import { CreateApiKeyModal, CreatedApiKeyModal } from "./ApiKeyCreationDialogs";

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
  const {
    data: roles = [],
    isLoading: rolesLoading,
    isFetching: rolesFetching,
    isError: rolesLoadFailed,
    refetch: refetchRoles,
  } = useOrganizationRoles(organizationId);
  const deleteMutation = useDeleteAPIKey(organizationId);
  const form = useCreateApiKeyForm(organizationId, canCreate);

  const assignableRoles = useMemo(() => {
    const reserved = new Set(["org_owner", "org_viewer", "org_admin"]);
    const customRoles = roles
      .flatMap((role) => {
        const name = role.metadata?.name;
        if (!name || name !== name.trim() || reserved.has(name)) return [];
        return [{ name, label: role.spec?.displayName?.trim() || name }];
      })
      .sort((a, b) => a.label.localeCompare(b.label) || a.name.localeCompare(b.name));

    return [{ name: "org_viewer", label: "Viewer" }, { name: "org_admin", label: "Admin" }, ...customRoles];
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
      <CreateApiKeyModal
        form={form}
        canvases={canvases}
        assignableRoles={assignableRoles}
        roleQueryStatus={{
          isLoading: rolesLoading,
          isFetching: rolesFetching,
          hasError: rolesLoadFailed,
          onRetry: () => void refetchRoles(),
        }}
      />
      <CreatedApiKeyModal form={form} />
    </div>
  );
}
