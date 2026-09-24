import { preferredFactoryScope, useCanvasFactoryScope } from "@/hooks/useCanvasFactoryScope";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import {
  defaultByokRunnerModel,
  SELECTABLE_LLM_SOURCE_BYOK,
  selectableLLMModelsForProvider,
} from "@/lib/selectableLLMModels";

/** Allowlisted model that should replace an empty value or a Claude alias such as "sonnet". */
export function useByokRunnerModelDefault(args: {
  enabled: boolean;
  organizationId: string | undefined;
  factoryId?: string;
  provider: string;
  current: string;
}): string | undefined {
  const canvasScope = useCanvasFactoryScope(args.organizationId);
  const { factoryId, waitingForCanvas } = preferredFactoryScope(args.factoryId, canvasScope);
  const query = useSelectableLLMModels(args.organizationId, {
    factoryId,
    sources: [SELECTABLE_LLM_SOURCE_BYOK],
    enabled: args.enabled && Boolean(args.organizationId) && !waitingForCanvas,
  });
  if (!args.enabled || waitingForCanvas || query.isLoading || query.isError) {
    return undefined;
  }
  const modelIds = selectableLLMModelsForProvider(query.data ?? [], args.provider).map((model) => model.model.id);
  return defaultByokRunnerModel(args.current, args.provider, modelIds);
}
