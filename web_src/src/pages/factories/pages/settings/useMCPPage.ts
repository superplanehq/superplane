import { useState } from "react";
import { useSearchParams } from "react-router";

import type { FactoriesFactoryAgentResource } from "@/api-client";
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
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import type { AgentResourceConnectionDraft } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { catalogEntryForResource, catalogOAuthResourceForEntry, type MCPCatalogEntry } from "./mcpCatalog";
import { bearerAuthorizationValue, catalogHeaderSecretName, MCP_HEADER_SECRET_KEY } from "./mcpHeaderAuth";

function useMCPAddDialog() {
  const [searchParams, setSearchParams] = useSearchParams();
  const addPickerOpen = searchParams.get("dialog") === "add";

  const setAddPickerOpen = (open: boolean) => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (open) {
          next.set("dialog", "add");
        } else {
          next.delete("dialog");
        }
        return next;
      },
      { replace: true },
    );
  };

  return { addPickerOpen, setAddPickerOpen };
}

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

export function useMCPPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const { addPickerOpen, setAddPickerOpen } = useMCPAddDialog();
  const [catalogEntry, setCatalogEntry] = useState<MCPCatalogEntry | undefined>();
  const [connectionOpen, setConnectionOpen] = useState(false);
  const [editResource, setEditResource] = useState<FactoriesFactoryAgentResource | undefined>();
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactoryAgentResource | undefined>();
  const connections = useFactoryAgentResources(organizationId, factoryId, "KIND_MCP_SERVER");
  const mutations = useMCPMutations(organizationId, factoryId);
  const closeConnection = () => {
    setEditResource(undefined);
    setConnectionOpen(false);
  };
  const closeCatalogSetup = () => {
    setCatalogEntry(undefined);
    setEditResource(undefined);
  };

  return {
    organizationId,
    factoryId,
    factory,
    canUpdate,
    addPickerOpen,
    connectionOpen: (connectionOpen || Boolean(editResource)) && !catalogEntry,
    catalogSetupOpen: Boolean(catalogEntry),
    catalogEntry,
    editResource,
    pendingDelete,
    connections,
    isSaving:
      mutations.createResource.isPending ||
      mutations.updateResource.isPending ||
      mutations.createSecret.isPending ||
      mutations.startOAuth.isPending,
    isDeleting: mutations.deleteResource.isPending,
    setAddPickerOpen,
    openCatalogEntry: (entry?: MCPCatalogEntry) => {
      setAddPickerOpen(false);
      closeConnection();
      if (!entry) {
        setCatalogEntry(undefined);
        setConnectionOpen(true);
        return;
      }
      setEditResource(catalogOAuthResourceForEntry(connections.data ?? [], entry));
      setCatalogEntry(entry);
    },
    startOAuthRedirect: (resource: FactoriesFactoryAgentResource) => startOAuthRedirect(mutations, resource),
    saveConnection: (draft: AgentResourceConnectionDraft) =>
      saveConnection(mutations, editResource, draft, closeConnection),
    saveCatalogToken: (entry: MCPCatalogEntry, token: string) =>
      saveCatalogToken(mutations, entry, token, closeCatalogSetup),
    startCatalogOAuth: (entry: MCPCatalogEntry) =>
      startCatalogOAuth(mutations, entry, editResource ?? catalogOAuthResourceForEntry(connections.data ?? [], entry), {
        rememberResource: setEditResource,
        onRedirecting: closeCatalogSetup,
      }),
    disconnectResource: (resource: FactoriesFactoryAgentResource) => disconnectResource(mutations, resource),
    toggleEnabled: (resource: FactoriesFactoryAgentResource, enabled: boolean) =>
      toggleEnabled(mutations, resource, enabled),
    toggleTools: (resource: FactoriesFactoryAgentResource, disabledTools: string[]) =>
      toggleTools(mutations, resource, disabledTools),
    setEditResource: (resource?: FactoriesFactoryAgentResource) => {
      setAddPickerOpen(false);
      if (!resource) {
        closeCatalogSetup();
        closeConnection();
        return;
      }
      const catalog = catalogEntryForResource(resource);
      if (catalog?.auth === "AUTH_OAUTH") {
        setConnectionOpen(false);
        setEditResource(resource);
        setCatalogEntry(catalog);
        return;
      }
      setCatalogEntry(undefined);
      setEditResource(resource);
      setConnectionOpen(true);
    },
    setPendingDelete,
    closeConnection,
    closeCatalogSetup,
    confirmDelete: () => confirmDelete(mutations, pendingDelete),
  };
}

type MCPMutations = {
  createResource: ReturnType<typeof useCreateFactoryAgentResource>;
  updateResource: ReturnType<typeof useUpdateFactoryAgentResource>;
  deleteResource: ReturnType<typeof useDeleteFactoryAgentResource>;
  startOAuth: ReturnType<typeof useStartFactoryAgentResourceOAuth>;
  disconnectOAuth: ReturnType<typeof useDisconnectFactoryAgentResourceOAuth>;
  createSecret: ReturnType<typeof useCreateSecret>;
};

async function startOAuthRedirect(mutations: MCPMutations, resource: FactoriesFactoryAgentResource) {
  if (!resource.id) {
    return;
  }
  try {
    const result = await mutations.startOAuth.mutateAsync(resource.id);
    if (result.authorizationUrl) {
      window.location.assign(result.authorizationUrl);
    }
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.connectFailed));
  }
}

async function startCatalogOAuth(
  mutations: MCPMutations,
  entry: MCPCatalogEntry,
  existing: FactoriesFactoryAgentResource | undefined,
  callbacks: {
    rememberResource: (resource: FactoriesFactoryAgentResource) => void;
    onRedirecting: () => void;
  },
) {
  let resource = existing;
  try {
    if (!resource?.id) {
      resource = await mutations.createResource.mutateAsync({
        kind: "KIND_MCP_SERVER",
        name: entry.name,
        enabled: true,
        url: entry.url,
        auth: "AUTH_OAUTH",
        headers: [],
      });
      callbacks.rememberResource(resource);
    }
    if (!resource?.id) {
      return;
    }
    const result = await mutations.startOAuth.mutateAsync(resource.id);
    if (!result.authorizationUrl) {
      return;
    }
    callbacks.onRedirecting();
    window.location.assign(result.authorizationUrl);
  } catch (error) {
    showErrorToast(
      getApiErrorMessage(error, resource?.id ? AGENT_RESOURCES_COPY.connectFailed : AGENT_RESOURCES_COPY.createFailed),
    );
    throw error;
  }
}

async function saveCatalogToken(mutations: MCPMutations, entry: MCPCatalogEntry, token: string, onSaved: () => void) {
  try {
    const secretName = catalogHeaderSecretName(entry.name, crypto.randomUUID());
    const secretResult = await mutations.createSecret.mutateAsync({
      name: secretName,
      environmentVariables: [{ name: MCP_HEADER_SECRET_KEY, value: bearerAuthorizationValue(token) }],
    });
    await mutations.createResource.mutateAsync({
      kind: "KIND_MCP_SERVER",
      name: entry.name,
      enabled: true,
      url: entry.url,
      auth: "AUTH_HEADERS",
      headers: [
        {
          name: entry.headerName ?? "Authorization",
          secretName: secretResult.data?.secret?.metadata?.name ?? secretName,
          secretKey: MCP_HEADER_SECRET_KEY,
        },
      ],
    });
    showSuccessToast(AGENT_RESOURCES_COPY.created);
    onSaved();
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.createFailed));
    throw error;
  }
}

async function saveConnection(
  mutations: MCPMutations,
  editResource: FactoriesFactoryAgentResource | undefined,
  draft: AgentResourceConnectionDraft,
  onSaved: () => void,
) {
  try {
    if (editResource?.id) {
      await mutations.updateResource.mutateAsync({
        resourceId: editResource.id,
        name: draft.name,
        url: draft.url,
        auth: draft.auth,
        headers: draft.headers,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.updated);
      onSaved();
      return;
    }
    await mutations.createResource.mutateAsync({
      kind: "KIND_MCP_SERVER",
      name: draft.name,
      enabled: true,
      url: draft.url,
      auth: draft.auth,
      headers: draft.headers,
    });
    showSuccessToast(AGENT_RESOURCES_COPY.created);
    onSaved();
  } catch (error) {
    showErrorToast(
      getApiErrorMessage(error, editResource ? AGENT_RESOURCES_COPY.updateFailed : AGENT_RESOURCES_COPY.createFailed),
    );
    throw error;
  }
}

function disconnectResource(mutations: MCPMutations, resource: FactoriesFactoryAgentResource) {
  if (!resource.id) {
    return;
  }
  void mutations.disconnectOAuth.mutateAsync(resource.id).then(
    () => showSuccessToast(AGENT_RESOURCES_COPY.disconnected),
    (error) => showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.disconnectFailed)),
  );
}

function toggleEnabled(mutations: MCPMutations, resource: FactoriesFactoryAgentResource, enabled: boolean) {
  if (!resource.id) {
    return;
  }
  void mutations.updateResource.mutateAsync({ resourceId: resource.id, enabled }).catch((error) => {
    showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.updateFailed));
  });
}

const mcpToolUpdateQueues = new Map<string, Promise<unknown>>();

function toggleTools(mutations: MCPMutations, resource: FactoriesFactoryAgentResource, disabledTools: string[]) {
  if (!resource.id) {
    return;
  }
  const resourceId = resource.id;
  const previous = mcpToolUpdateQueues.get(resourceId) ?? Promise.resolve();
  const next = previous
    .catch(() => undefined)
    .then(() =>
      mutations.updateResource.mutateAsync({ resourceId, disabledTools, replaceDisabledTools: true }).catch((error) => {
        showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.updateFailed));
      }),
    );
  mcpToolUpdateQueues.set(resourceId, next);
}

async function confirmDelete(mutations: MCPMutations, pendingDelete?: FactoriesFactoryAgentResource) {
  if (!pendingDelete?.id) {
    return;
  }
  try {
    await mutations.deleteResource.mutateAsync(pendingDelete.id);
    showSuccessToast(AGENT_RESOURCES_COPY.deleted);
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.deleteFailed));
    throw error;
  }
}
