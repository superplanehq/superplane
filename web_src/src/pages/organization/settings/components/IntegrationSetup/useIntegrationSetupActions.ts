import type { Dispatch, SetStateAction } from "react";
import { useCallback } from "react";
import type {
  IntegrationSetupMetadata,
  IntegrationSetupMutations,
  IntegrationSetupProgress,
  IntegrationSetupRoute,
  IntegrationSetupState,
} from "./useIntegrationSetupController";
import { showErrorToast } from "@/lib/toast";
import { getGroupToggleState } from "./lib";

interface SetupActionsParams {
  route: IntegrationSetupRoute;
  state: IntegrationSetupState;
  progress: IntegrationSetupProgress;
  metadata: IntegrationSetupMetadata;
  mutations: IntegrationSetupMutations;
}

export function useIntegrationSetupActions({ route, state, progress, metadata, mutations }: SetupActionsParams) {
  const handleCreateIntegration = useCreateIntegrationHandler({ route, state, metadata, mutations });
  const capabilityToggles = useCapabilitySelectionActions({
    createMutation: mutations.createMutation,
    setSelectedCapabilities: state.setSelectedCapabilities,
  });
  const handleStepInputChange = useStepInputChange(state.setStepInputs);
  const handleSubmitCurrentStep = useSubmitCurrentStepAction({ state, progress, mutations });
  const handleRevertCurrentStep = useRevertCurrentStepAction({ state, progress, mutations });
  const handleDiscardIntegration = useDiscardIntegrationAction({ route, state, mutations });
  const handleSetupStepBack = useSetupStepBackAction({
    state,
    progress,
    mutations,
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
  metadata: IntegrationSetupMetadata;
  mutations: Pick<IntegrationSetupMutations, "createMutation">;
}

function useCreateIntegrationHandler({ route, state, metadata, mutations }: CreateIntegrationHandlerParams) {
  return useCallback(async () => {
    const trimmedName = state.instanceName.trim();
    if (!trimmedName) {
      showErrorToast("Integration name is required");
      return;
    }

    if (metadata.integrationCapabilities.length > 0 && state.selectedCapabilities.size === 0) {
      showErrorToast("Select at least one capability");
      return;
    }

    try {
      const response = await mutations.createMutation.mutateAsync({
        integrationName: route.integrationName,
        name: trimmedName,
        capabilities: Array.from(state.selectedCapabilities),
      });
      state.setCreatedIntegration(response.data?.integration || null);
      state.setStepInputs({});
    } catch {
      // Error is shown by inline alert.
    }
  }, [route.integrationName, state, metadata.integrationCapabilities, mutations.createMutation]);
}

interface CapabilitySelectionActionsParams {
  createMutation: IntegrationSetupMutations["createMutation"];
  setSelectedCapabilities: Dispatch<SetStateAction<Set<string>>>;
}

function useCapabilitySelectionActions({ createMutation, setSelectedCapabilities }: CapabilitySelectionActionsParams) {
  const toggleCapabilitySelection = useCallback(
    (capabilityName: string) => {
      if (createMutation.isPending) {
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
    [createMutation, setSelectedCapabilities],
  );

  const toggleCapabilityGroup = useCallback(
    (capabilityNames: string[]) => {
      if (createMutation.isPending || capabilityNames.length === 0) {
        return;
      }

      setSelectedCapabilities((previous) => {
        const state = getGroupToggleState(capabilityNames, previous);
        const next = new Set(previous);
        if (state === "all") {
          capabilityNames.forEach((name) => next.delete(name));
        } else {
          capabilityNames.forEach((name) => next.add(name));
        }
        return next;
      });
    },
    [createMutation, setSelectedCapabilities],
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

function useSubmitCurrentStepAction({ state, progress, mutations }: CurrentStepActionParams) {
  return useCallback(async () => {
    const integrationId = state.createdIntegration?.metadata?.id;
    if (!integrationId || !progress.currentStep) {
      return;
    }

    if (progress.currentStep.type !== "DONE" && !progress.currentStep.name) {
      return;
    }

    try {
      const response = await mutations.submitStepMutation.mutateAsync({
        integrationId,
        inputs: progress.currentStep.type === "INPUTS" ? state.stepInputs : undefined,
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
    if (!window.confirm("Discard this integration? It will be removed and this cannot be undone.")) {
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
  mutations: Pick<IntegrationSetupMutations, "deleteIntegrationMutation">;
  handleRevertCurrentStep: () => Promise<void>;
}

function useSetupStepBackAction({ state, progress, mutations, handleRevertCurrentStep }: SetupStepBackActionParams) {
  return useCallback(async () => {
    if (!progress.currentStep || !state.createdIntegration?.metadata?.id) {
      return;
    }

    if (progress.canRevertCurrentStep) {
      await handleRevertCurrentStep();
      return;
    }

    if (
      !window.confirm("Remove this partially configured integration? You'll return to naming and capability selection.")
    ) {
      return;
    }

    try {
      await mutations.deleteIntegrationMutation.mutateAsync({
        integrationName: state.createdIntegration?.metadata?.integrationName ?? "",
      });
      state.setCreatedIntegration(null);
      state.setStepInputs({});
    } catch {
      showErrorToast("Failed to remove integration");
    }
  }, [
    state,
    progress.currentStep,
    progress.canRevertCurrentStep,
    mutations.deleteIntegrationMutation,
    handleRevertCurrentStep,
  ]);
}
