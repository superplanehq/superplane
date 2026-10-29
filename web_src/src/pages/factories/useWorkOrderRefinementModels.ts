import { useFactoryAutomations } from "@/hooks/useFactoryData";
import { useComponent } from "@/hooks/useComponentData";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { HOSTED_MODEL_ALL_PROVIDERS, shortClaudeModelLabel } from "@/lib/hostedLLMModels";
import {
  byokRunnerModelOptions,
  hostedSelectableLLMModelKey,
  normalizeSuperPlaneModelValue,
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_SUBSCRIPTION,
  SELECTABLE_LLM_SOURCE_HOSTED,
  selectableLLMModelsForProvider,
} from "@/lib/selectableLLMModels";
import { findBacklogAutomationApp } from "./lib/linePhaseRuns";
import { useColumnCanvasAgentEditor } from "./pages/useColumnCanvasAgentEditor";
import { planningReviewModelUsedField } from "./pages/planningReviewRunnerFields";
import type { WorkOrderRefinementModelSelection } from "./WorkOrderRefinementModelSelect";
import type { ConfigurationField } from "@/api-client";
import type { SelectableLLMModel } from "@/lib/selectableLLMModels";

export function useWorkOrderRefinementModels(
  organizationId: string,
  factoryId: string,
  orderId?: string,
): WorkOrderRefinementModelSelection {
  const apps = useFactoryAutomations(organizationId, factoryId);
  const canvasId = findBacklogAutomationApp(apps.data ?? [])?.id;
  const editor = useColumnCanvasAgentEditor(organizationId, canvasId, { preferredAgentNodeId: "refine-task" });
  const node = refinementNode(editor.agentNode);
  const component = node?.component ?? "";
  const action = useComponent(organizationId, component);
  const field = planningReviewModelUsedField(component, action.data?.configuration);
  const provider = field?.typeOptions?.hostedModel?.provider ?? "";
  const hosted = provider === HOSTED_MODEL_ALL_PROVIDERS;
  const configuration = refinementConfiguration(node);
  const models = useSelectableLLMModels(organizationId, {
    factoryId,
    orderId,
    sources: [refinementCatalogSource(configuration, hosted)],
    enabled: Boolean(node && provider),
  });
  const usage = useOrganizationWorkspaceUsage(organizationId);
  const current = modelValue(configuration.model);
  const isLoading = [
    apps.isPending,
    editor.isLoading,
    action.isLoading,
    pickerLoading(provider, models.isLoading, usage.isLoading),
  ].some(Boolean);
  if ([!node, apps.isError, action.isError, models.isError].some(Boolean)) {
    return { defaultModel: current, options: [], isLoading };
  }
  const selection = refinementModelSelection(field, configuration, models.data ?? [], usage.data);
  return { ...selection, isLoading };
}

function refinementNode<T extends { id?: string }>(node: T | undefined): T | undefined {
  return node?.id === "refine-task" ? node : undefined;
}

function refinementConfiguration(node: { configuration?: Record<string, unknown> } | undefined) {
  return node?.configuration ?? {};
}

function pickerLoading(provider: string, modelsLoading: boolean, usageLoading: boolean): boolean {
  if (!provider) {
    return false;
  }
  return modelsLoading || (provider === HOSTED_MODEL_ALL_PROVIDERS && usageLoading);
}

function modelValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function refinementModelSelection(
  field: ConfigurationField | undefined,
  configuration: Record<string, unknown>,
  models: SelectableLLMModel[],
  usage: { defaultHostedProvider?: string; defaultHostedModel?: string } | undefined,
): WorkOrderRefinementModelSelection {
  if (refinementCatalogSource(configuration, false) === SELECTABLE_LLM_SOURCE_SUBSCRIPTION) {
    return { defaultModel: "", options: models.map((model) => ({ value: model.model.id, label: model.label })) };
  }
  const current = modelValue(configuration.model);
  const provider = field?.typeOptions?.hostedModel?.provider ?? "";
  if (!provider) {
    return staticModelSelection(field, current);
  }
  const hosted = provider === HOSTED_MODEL_ALL_PROVIDERS;
  if (hosted) {
    const options = models.map((model) => ({ value: model.key, label: model.label }));
    const defaultModel =
      normalizeSuperPlaneModelValue(current) ||
      hostedSelectableLLMModelKey(usage?.defaultHostedProvider ?? "", usage?.defaultHostedModel ?? "");
    return { defaultModel, options };
  }
  return byokModelSelection(current, configuration.llmProvider === "custom" ? "custom" : provider, models);
}

function staticModelSelection(
  field: ConfigurationField | undefined,
  current: string,
): WorkOrderRefinementModelSelection {
  return {
    defaultModel: current,
    options:
      field?.typeOptions?.select?.options
        ?.map((option) => ({ value: option.value ?? "", label: option.label ?? option.value ?? "" }))
        .filter((option) => option.value) ?? [],
  };
}

function byokModelSelection(
  current: string,
  provider: string,
  models: SelectableLLMModel[],
): WorkOrderRefinementModelSelection {
  const providerModels = selectableLLMModelsForProvider(models, provider);
  return {
    defaultModel: current,
    options: providerModels.length
      ? byokRunnerModelOptions(providerModels, current).map((option) => ({
          ...option,
          label: shortClaudeModelLabel(option.value) ?? option.label,
        }))
      : [],
  };
}

function refinementCatalogSource(configuration: Record<string, unknown>, hosted: boolean) {
  const credentials = configuration.credentials as { source?: string } | undefined;
  if (credentials?.source === "subscription") return SELECTABLE_LLM_SOURCE_SUBSCRIPTION;
  return hosted ? SELECTABLE_LLM_SOURCE_HOSTED : SELECTABLE_LLM_SOURCE_BYOK;
}
