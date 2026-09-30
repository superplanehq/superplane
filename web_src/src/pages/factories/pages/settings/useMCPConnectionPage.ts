import { useState } from "react";
import { useNavigate, useParams } from "react-router";

import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useDisconnectFactoryAgentResourceOAuth,
  useFactoryAgentResources,
  useStartFactoryAgentResourceOAuth,
  useUpdateFactoryAgentResource,
} from "@/hooks/useFactoryAgentResources";
import { useCreateSecret } from "@/hooks/useSecrets";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import type { AgentResourceConnectionDraft } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { catalogEntryForResource, type MCPCatalogEntry } from "./mcpCatalog";
import { mcpConnectionDisplayName } from "./mcpConnectionDisplay";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import {
  confirmMCPDelete,
  disconnectMCPResource,
  saveCatalogToken,
  saveMCPConnection,
  startMCPOAuthRedirect,
  startCatalogOAuth,
  toggleMCPEnabled,
  toggleMCPTools,
} from "./useMCPPage";

function useMCPMutations(organizationId: string, factoryId: string) {
  return {
    createResource: useCreateFactoryAgentResource(organizationId, factoryId),
    updateResource: useUpdateFactoryAgentResource(organizationId, factoryId),
    deleteResource: useDeleteFactoryAgentResource(organizationId, factoryId),
    startOAuth: useStartFactoryAgentResourceOAuth(organizationId, factoryId),
    disconnectOAuth: useDisconnectFactoryAgentResourceOAuth(organizationId, factoryId),
    createSecret: useCreateSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION"),
  };
}

export function useMCPConnectionPage() {
  const { resourceId = "" } = useParams<{ resourceId: string }>();
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const factoryKey = factory.key ?? "";
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const navigate = useNavigate();
  const connections = useFactoryAgentResources(organizationId, factoryId, "KIND_MCP_SERVER");
  const resource = connections.data?.find((entry) => entry.id === resourceId);
  const mutations = useMCPMutations(organizationId, factoryId);
  const [connectionOpen, setConnectionOpen] = useState(false);
  const [catalogEntry, setCatalogEntry] = useState<MCPCatalogEntry | undefined>();
  const [pendingDelete, setPendingDelete] = useState(false);

  const listPath = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "agent");

  const openEdit = () => {
    if (!resource) {
      return;
    }
    const catalog = catalogEntryForResource(resource);
    if (catalog?.auth === "AUTH_OAUTH") {
      setConnectionOpen(false);
      setCatalogEntry(catalog);
      return;
    }
    setCatalogEntry(undefined);
    setConnectionOpen(true);
  };

  const closeEdit = () => {
    setConnectionOpen(false);
    setCatalogEntry(undefined);
  };

  return {
    organizationId,
    factoryId,
    factory,
    canUpdate,
    listPath,
    resource,
    displayName: resource ? mcpConnectionDisplayName(resource) : AGENT_RESOURCES_COPY.unnamedResource,
    isLoading: connections.isLoading,
    connectionOpen,
    catalogSetupOpen: Boolean(catalogEntry),
    catalogEntry,
    pendingDelete,
    isSaving: mutations.updateResource.isPending || mutations.createSecret.isPending,
    isDeleting: mutations.deleteResource.isPending,
    navigateToList: () => navigate(listPath),
    openEdit,
    closeEdit,
    closeCatalogSetup: () => setCatalogEntry(undefined),
    saveConnection: (draft: AgentResourceConnectionDraft) =>
      resource ? saveMCPConnection(mutations, connections.data ?? [], resource, draft, closeEdit) : Promise.resolve(),
    startCatalogOAuth: (entry: MCPCatalogEntry) =>
      startCatalogOAuth(mutations, entry, resource, {
        rememberResource: () => undefined,
        onRedirecting: () => setCatalogEntry(undefined),
      }),
    saveCatalogToken: (entry: MCPCatalogEntry, token: string) =>
      saveCatalogToken(mutations, connections.data ?? [], entry, token, () => setCatalogEntry(undefined)),
    startOAuthRedirect: () => {
      if (resource) {
        void startMCPOAuthRedirect(mutations, resource);
      }
    },
    disconnect: () => {
      if (resource) {
        disconnectMCPResource(mutations, resource);
      }
    },
    toggleEnabled: (enabled: boolean) => {
      if (resource) {
        toggleMCPEnabled(mutations, resource, enabled);
      }
    },
    toggleTools: (disabledTools: string[]) => {
      if (resource) {
        toggleMCPTools(mutations, resource, disabledTools);
      }
    },
    setPendingDelete,
    confirmDelete: async () => {
      if (!resource) {
        return;
      }
      try {
        await confirmMCPDelete(mutations, resource);
        navigate(listPath);
      } catch {
        // toast handled in confirmMCPDelete
      }
    },
  };
}
