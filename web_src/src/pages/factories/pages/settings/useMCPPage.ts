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
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import type { AgentResourceConnectionDraft } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import type { MCPCatalogEntry } from "./mcpCatalog";

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
  };
}

export function useMCPPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const { addPickerOpen, setAddPickerOpen } = useMCPAddDialog();
  const [catalogDefaults, setCatalogDefaults] = useState<Partial<AgentResourceConnectionDraft> | undefined>();
  const [connectionOpen, setConnectionOpen] = useState(false);
  const [editResource, setEditResource] = useState<FactoriesFactoryAgentResource | undefined>();
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactoryAgentResource | undefined>();
  const connections = useFactoryAgentResources(organizationId, factoryId, "KIND_MCP_SERVER");
  const mutations = useMCPMutations(organizationId, factoryId);
  const closeConnection = () => {
    setEditResource(undefined);
    setConnectionOpen(false);
    setCatalogDefaults(undefined);
  };

  return {
    organizationId,
    factoryId,
    factory,
    canUpdate,
    addPickerOpen,
    connectionOpen: connectionOpen || Boolean(editResource),
    catalogDefaults,
    editResource,
    pendingDelete,
    connections,
    isSaving: mutations.createResource.isPending || mutations.updateResource.isPending,
    isDeleting: mutations.deleteResource.isPending,
    setAddPickerOpen,
    openCatalogEntry: (entry?: MCPCatalogEntry) => {
      setAddPickerOpen(false);
      closeConnection();
      setCatalogDefaults(entry ? { name: entry.name, url: entry.url, auth: entry.auth } : undefined);
      setConnectionOpen(true);
    },
    startOAuthRedirect: (resource: FactoriesFactoryAgentResource) => startOAuthRedirect(mutations, resource),
    saveConnection: (draft: AgentResourceConnectionDraft) =>
      saveConnection(mutations, editResource, draft, closeConnection),
    disconnectResource: (resource: FactoriesFactoryAgentResource) => disconnectResource(mutations, resource),
    toggleEnabled: (resource: FactoriesFactoryAgentResource, enabled: boolean) =>
      toggleEnabled(mutations, resource, enabled),
    toggleTools: (resource: FactoriesFactoryAgentResource, disabledTools: string[]) =>
      toggleTools(mutations, resource, disabledTools),
    setEditResource: (resource?: FactoriesFactoryAgentResource) => {
      setCatalogDefaults(undefined);
      setEditResource(resource);
      setConnectionOpen(Boolean(resource));
    },
    setPendingDelete,
    closeConnection,
    confirmDelete: () => confirmDelete(mutations, pendingDelete),
  };
}

type MCPMutations = {
  createResource: ReturnType<typeof useCreateFactoryAgentResource>;
  updateResource: ReturnType<typeof useUpdateFactoryAgentResource>;
  deleteResource: ReturnType<typeof useDeleteFactoryAgentResource>;
  startOAuth: ReturnType<typeof useStartFactoryAgentResourceOAuth>;
  disconnectOAuth: ReturnType<typeof useDisconnectFactoryAgentResourceOAuth>;
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

function toggleTools(mutations: MCPMutations, resource: FactoriesFactoryAgentResource, disabledTools: string[]) {
  if (!resource.id) {
    return;
  }
  void mutations.updateResource
    .mutateAsync({ resourceId: resource.id, disabledTools, setDisabledTools: true })
    .catch((error) => {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.updateFailed));
    });
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
