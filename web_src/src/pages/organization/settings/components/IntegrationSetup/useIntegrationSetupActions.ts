import type { Dispatch, SetStateAction } from "react";
import { useCallback } from "react";
import type {
  IntegrationSetupMutations,
  IntegrationSetupProgress,
  IntegrationSetupRoute,
  IntegrationSetupState,
} from "./useIntegrationSetupController";
import { showErrorToast } from "@/lib/toast";
import { getGroupToggleState } from "./lib";
import type { IntegrationSetupStepDefinitionType } from "@/api-client";

interface SetupActionsParams {
  route: IntegrationSetupRoute;
  state: IntegrationSetupState;
  progress: IntegrationSetupProgress;
  mutations: IntegrationSetupMutations;
}

export function useIntegrationSetupActions({ route, state, progress, mutations }: SetupActionsParams) {
  const handleCreateIntegration = useCreateIntegrationHandler({ route, state, mutations });
  const capabilityToggles = useCapabilitySelectionActions({
    createMutation: mutations.createMutation,
    submitStepMutation: mutations.submitStepMutation,
    revertStepMutation: mutations.revertStepMutation,
    setSelectedCapabilities: state.setSelectedCapabilities,
  });
  const handleStepInputChange = useStepInputChange(state.setStepInputs);
  const handleSubmitCurrentStep = useSubmitCurrentStepAction({ state, progress, mutations });
  const handleRevertCurrentStep = useRevertCurrentStepAction({ state, progress, mutations });
  const handleDiscardIntegration = useDiscardIntegrationAction({ route, state, mutations });
  const handleSetupStepBack = useSetupStepBackAction({
    state,
    progress,
    handleRevertCurrentStep,
  });

  return {
    handleCreateIntegration,
    handleStepInputChange,
    handleSubmitCurrentStep,
    handleDiscardIntegration,
    handleSetupStepBack,
    ...capabilityToggles,
  };
}

interface CreateIntegrationHandlerParams {
  route: IntegrationSetupRoute;
  state: IntegrationSetupState;
  mutations: Pick<IntegrationSetupMutations, "createMutation">;
}

function useCreateIntegrationHandler({ route, state, mutations }: CreateIntegrationHandlerParams) {
  return useCallback(async () => {
    const trimmedName = state.instanceName.trim();
    if (!trimmedName) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      const response = await mutations.createMutation.mutateAsync({
        integrationName: route.integrationName,
        name: trimmedName,
      });
      state.setCreatedIntegration(response.data?.integration || null);
      state.setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  }, [route.integrationName, state, mutations.createMutation]);
}

interface CapabilitySelectionActionsParams {
  createMutation: IntegrationSetupMutations["createMutation"];
  submitStepMutation: IntegrationSetupMutations["submitStepMutation"];
  revertStepMutation: IntegrationSetupMutations["revertStepMutation"];
  setSelectedCapabilities: Dispatch<SetStateAction<Set<string>>>;
}

function useCapabilitySelectionActions({
  createMutation,
  submitStepMutation,
  revertStepMutation,
  setSelectedCapabilities,
}: CapabilitySelectionActionsParams) {
  const toggleCapabilitySelection = useCallback(
    (capabilityName: string) => {
      if (createMutation.isPending || submitStepMutation.isPending || revertStepMutation.isPending) {
        return;
      }

      setSelectedCapabilities((current) => {
        const next = new Set(current);
        if (next.has(capabilityName)) {
          next.delete(capabilityName);
        } else {
          next.add(capabilityName);
        }
        return next;
      });
    },
    [createMutation.isPending, submitStepMutation.isPending, revertStepMutation.isPending, setSelectedCapabilities],
  );

  const toggleCapabilityGroup = useCallback(
    (capabilityNames: string[]) => {
      if (
        createMutation.isPending ||
        submitStepMutation.isPending ||
        revertStepMutation.isPending ||
        capabilityNames.length === 0
      ) {
        return;
      }

      setSelectedCapabilities((previous) => {
        const groupState = getGroupToggleState(capabilityNames, previous);
        const next = new Set(previous);
        if (groupState === "all") {
          capabilityNames.forEach((name) => next.delete(name));
        } else {
          capabilityNames.forEach((name) => next.add(name));
        }
        return next;
      });
    },
    [createMutation.isPending, submitStepMutation.isPending, revertStepMutation.isPending, setSelectedCapabilities],
  );

  return {
    toggleCapabilitySelection,
    toggleCapabilityGroup,
  };
}

function useStepInputChange(setStepInputs: Dispatch<SetStateAction<Record<string, unknown>>>) {
  return useCallback(
    (fieldName: string, value: unknown) => {
      setStepInputs((currentValues) => ({ ...currentValues, [fieldName]: value }));
    },
    [setStepInputs],
  );
}

interface CurrentStepActionParams {
  state: IntegrationSetupState;
  progress: IntegrationSetupProgress;
  mutations: Pick<IntegrationSetupMutations, "submitStepMutation" | "revertStepMutation">;
}

function submitStepBody(type: IntegrationSetupStepDefinitionType, state: IntegrationSetupState) {
  switch (type) {
    case "INPUTS":
      return {
        inputs: state.stepInputs,
      };

    case "CAPABILITY_SELECTION":
      return {
        capabilities: Array.from(state.selectedCapabilities),
      };
  }

  return {};
}

function useSubmitCurrentStepAction({ state, progress, mutations }: CurrentStepActionParams) {
  return useCallback(async () => {
    const integrationId = state.createdIntegration?.metadata?.id;
    if (!integrationId || !progress.currentStep) {
      return;
    }

    if (progress.currentStep.type !== "DONE" && !progress.currentStep.name) {
      return;
    }

    if (progress.currentStep.type === "CAPABILITY_SELECTION") {
      const offered = progress.currentStep.capabilities ?? [];
      if (offered.length > 0 && state.selectedCapabilities.size === 0) {
        showErrorToast("Select at least one capability");
        return;
      }
    }

    try {
      const response = await mutations.submitStepMutation.mutateAsync({
        integrationId,
        ...submitStepBody(progress.currentStep.type!, state),
      });
      state.setCreatedIntegration(response.data?.integration || null);
      state.setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  }, [state, progress.currentStep, mutations.submitStepMutation]);
}

function useRevertCurrentStepAction({ state, progress, mutations }: CurrentStepActionParams) {
  return useCallback(async () => {
    const integrationId = state.createdIntegration?.metadata?.id;
    if (!integrationId || !progress.currentStep) {
      return;
    }

    if (progress.currentStep.type !== "DONE" && !progress.currentStep.name) {
      return;
    }

    try {
      const response = await mutations.revertStepMutation.mutateAsync({ integrationId });
      state.setCreatedIntegration(response.data?.integration || null);
      state.setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  }, [state, progress.currentStep, mutations.revertStepMutation]);
}

interface DiscardIntegrationActionParams {
  route: IntegrationSetupRoute;
  state: IntegrationSetupState;
  mutations: Pick<IntegrationSetupMutations, "deleteIntegrationMutation">;
}

function useDiscardIntegrationAction({ route, state, mutations }: DiscardIntegrationActionParams) {
  return useCallback(async () => {
    const integrationId = state.createdIntegration?.metadata?.id;
    if (!integrationId) {
      return;
    }
    if (!window.confirm("Delete this integration? It will be removed and this cannot be undone.")) {
      return;
    }
    try {
      await mutations.deleteIntegrationMutation.mutateAsync({
        integrationName: state.createdIntegration?.metadata?.integrationName ?? "",
      });
      state.setCreatedIntegration(null);
      route.navigate(route.integrationsHref);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  }, [route, state, mutations.deleteIntegrationMutation]);
}

interface SetupStepBackActionParams {
  state: IntegrationSetupState;
  progress: IntegrationSetupProgress;
  handleRevertCurrentStep: () => Promise<void>;
}

function useSetupStepBackAction({ state, progress, handleRevertCurrentStep }: SetupStepBackActionParams) {
  return useCallback(async () => {
    if (!progress.currentStep || !state.createdIntegration?.metadata?.id) {
      return;
    }

    await handleRevertCurrentStep();
  }, [state, progress.currentStep, handleRevertCurrentStep]);
}
