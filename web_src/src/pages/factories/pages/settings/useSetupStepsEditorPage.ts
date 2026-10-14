import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useFactoryAgentResources,
  useUpdateFactoryAgentResource,
} from "@/hooks/useFactoryAgentResources";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryAgentPath, factoryRouteSegment } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

function parseCommands(text: string): string[] {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

export function useSetupStepsEditorPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const navigate = useNavigate();
  const setup = useFactoryAgentResources(organizationId, factoryId, "KIND_SETUP_COMMAND");
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const listPath = factoryAgentPath(organizationId, factoryRouteSegment(factory));

  const stored = (setup.data ?? [])[0];
  const storedText = (stored?.commands ?? []).join("\n");
  const [value, setValue] = useState("");
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (!setup.isLoading && !initialized) {
      setValue(storedText);
      setInitialized(true);
    }
  }, [setup.isLoading, storedText, initialized]);

  const isSaving = createResource.isPending || updateResource.isPending || deleteResource.isPending;
  const isDirty = parseCommands(value).join("\n") !== parseCommands(storedText).join("\n");

  const handleSave = async () => {
    const commands = parseCommands(value);
    try {
      if (!stored?.id) {
        if (commands.length === 0) {
          navigate(listPath);
          return;
        }
        await createResource.mutateAsync({ kind: "KIND_SETUP_COMMAND", name: "setup", enabled: true, commands });
      } else if (commands.length === 0) {
        await deleteResource.mutateAsync(stored.id);
      } else {
        await updateResource.mutateAsync({ resourceId: stored.id, commands, replaceCommands: true });
      }
      showSuccessToast(AGENT_RESOURCES_COPY.setupStepsSaved);
      navigate(listPath);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.setupStepsSaveFailed));
    }
  };

  return {
    factory,
    canUpdate,
    listPath,
    value,
    setupLoading: setup.isLoading,
    setupError: setup.isError,
    isSaving,
    isDirty,
    setValue,
    navigateToList: () => navigate(listPath),
    handleSave,
  };
}
