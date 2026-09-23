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
import type { AgentResourceSkillDraft } from "./AgentResourceSkillDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export type AgentResourcesTab = "connections" | "skills";

export function readAgentResourcesTab(searchParams: URLSearchParams): AgentResourcesTab {
  return searchParams.get("tab") === "skills" ? "skills" : "connections";
}

function useAgentResourceSearch() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = readAgentResourcesTab(searchParams);
  const addDialogOpen = searchParams.get("dialog") === "add";

  const setTab = (nextTab: string) => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (nextTab === "skills") {
          next.set("tab", "skills");
        } else {
          next.delete("tab");
        }
        next.delete("dialog");
        return next;
      },
      { replace: true },
    );
  };

  const setAddDialogOpen = (open: boolean) => {
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

  return { tab, addDialogOpen, setTab, setAddDialogOpen };
}

function useAgentResourceActions({
  organizationId,
  factoryId,
  editResource,
  pendingDelete,
  setEditResource,
  setAddDialogOpen,
}: {
  organizationId: string;
  factoryId: string;
  editResource?: FactoriesFactoryAgentResource;
  pendingDelete?: FactoriesFactoryAgentResource;
  setEditResource: (resource?: FactoriesFactoryAgentResource) => void;
  setAddDialogOpen: (open: boolean) => void;
}) {
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const startOAuth = useStartFactoryAgentResourceOAuth(organizationId, factoryId);
  const disconnectOAuth = useDisconnectFactoryAgentResourceOAuth(organizationId, factoryId);

  const startOAuthRedirect = async (resource: FactoriesFactoryAgentResource) => {
    if (!resource.id) {
      return;
    }
    try {
      const result = await startOAuth.mutateAsync(resource.id);
      if (result.authorizationUrl) {
        window.location.assign(result.authorizationUrl);
      }
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.connectFailed));
    }
  };

  const saveConnection = async (draft: AgentResourceConnectionDraft) => {
    try {
      if (editResource?.id) {
        await updateResource.mutateAsync({
          resourceId: editResource.id,
          name: draft.name,
          url: draft.url,
          auth: draft.auth,
          headers: draft.headers,
        });
        showSuccessToast(AGENT_RESOURCES_COPY.updated);
        setEditResource(undefined);
        return;
      }
      await createResource.mutateAsync({
        kind: "KIND_MCP_SERVER",
        name: draft.name,
        enabled: true,
        url: draft.url,
        auth: draft.auth,
        headers: draft.headers,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.created);
      setAddDialogOpen(false);
    } catch (error) {
      showErrorToast(
        getApiErrorMessage(error, editResource ? AGENT_RESOURCES_COPY.updateFailed : AGENT_RESOURCES_COPY.createFailed),
      );
      throw error;
    }
  };

  const saveSkill = async (draft: AgentResourceSkillDraft) => {
    try {
      if (editResource?.id) {
        await updateResource.mutateAsync({
          resourceId: editResource.id,
          name: draft.name,
          markdown: draft.markdown,
        });
        showSuccessToast(AGENT_RESOURCES_COPY.skillUpdated);
        setEditResource(undefined);
        return;
      }
      await createResource.mutateAsync({
        kind: "KIND_SKILL",
        name: draft.name,
        enabled: true,
        markdown: draft.markdown,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.skillCreated);
      setAddDialogOpen(false);
    } catch (error) {
      showErrorToast(
        getApiErrorMessage(
          error,
          editResource ? AGENT_RESOURCES_COPY.skillUpdateFailed : AGENT_RESOURCES_COPY.skillCreateFailed,
        ),
      );
      throw error;
    }
  };

  const disconnectResource = (resource: FactoriesFactoryAgentResource) => {
    if (!resource.id) {
      return;
    }
    void disconnectOAuth.mutateAsync(resource.id).then(
      () => showSuccessToast(AGENT_RESOURCES_COPY.disconnected),
      (error) => showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.disconnectFailed)),
    );
  };

  const toggleEnabled = (resource: FactoriesFactoryAgentResource, enabled: boolean) => {
    if (!resource.id) {
      return;
    }
    void updateResource.mutateAsync({ resourceId: resource.id, enabled }).catch((error) => {
      showErrorToast(
        getApiErrorMessage(
          error,
          resource.kind === "KIND_SKILL" ? AGENT_RESOURCES_COPY.skillUpdateFailed : AGENT_RESOURCES_COPY.updateFailed,
        ),
      );
    });
  };

  const confirmDelete = async () => {
    if (!pendingDelete?.id) {
      return;
    }
    const isSkill = pendingDelete.kind === "KIND_SKILL";
    try {
      await deleteResource.mutateAsync(pendingDelete.id);
      showSuccessToast(isSkill ? AGENT_RESOURCES_COPY.skillDeleted : AGENT_RESOURCES_COPY.deleted);
    } catch (error) {
      showErrorToast(
        getApiErrorMessage(error, isSkill ? AGENT_RESOURCES_COPY.skillDeleteFailed : AGENT_RESOURCES_COPY.deleteFailed),
      );
      throw error;
    }
  };

  return {
    isSaving: createResource.isPending || updateResource.isPending,
    isDeleting: deleteResource.isPending,
    startOAuthRedirect,
    saveConnection,
    saveSkill,
    disconnectResource,
    toggleEnabled,
    confirmDelete,
  };
}

export function useAgentResourcesPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const { tab, addDialogOpen, setTab, setAddDialogOpen } = useAgentResourceSearch();
  const [editResource, setEditResource] = useState<FactoriesFactoryAgentResource | undefined>();
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactoryAgentResource | undefined>();
  const connections = useFactoryAgentResources(organizationId, factoryId, "KIND_MCP_SERVER");
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");
  const actions = useAgentResourceActions({
    organizationId,
    factoryId,
    editResource,
    pendingDelete,
    setEditResource,
    setAddDialogOpen,
  });

  return {
    organizationId,
    factoryId,
    factory,
    canUpdate,
    tab,
    addDialogOpen,
    editResource,
    pendingDelete,
    connections,
    skills,
    setTab,
    setAddDialogOpen,
    setEditResource,
    setPendingDelete,
    ...actions,
  };
}
