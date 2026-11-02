import { useState } from "react";

import type { FactoriesFactoryLlmModel, FactoriesListFactoryLlmModelsResponse } from "@/api-client";
import { useFactoryLLMModels, useUpdateFactoryLLMModels } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { displayModelName } from "@/lib/hostedLLMModels";

import { ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";

export type WorkspaceModelPolicyScope = {
  organizationId: string;
  factoryId: string;
  provider: string;
  fundingSource: "hosted" | "byok" | "subscription";
};
type ModelPolicy = { inheritParent: boolean; ids: string[] };

function idsOf(models: FactoriesFactoryLlmModel[]) {
  return models.map((model) => model.id ?? "").filter(Boolean);
}

function modelCatalog(response: FactoriesListFactoryLlmModelsResponse) {
  const parent = response.parent ?? [];
  const configured = response.configuredModels ?? response.selected ?? [];
  const availableIds = idsOf(parent);
  const configuredIds = idsOf(configured);
  const modelIds = [...new Set([...availableIds, ...configuredIds])];
  const models = [...parent, ...configured];
  const labels = Object.fromEntries(
    modelIds.map((id) => {
      const name = models.find((model) => model.id === id)?.name;
      return [id, name && name !== id ? name : displayModelName(id)];
    }),
  );
  return { availableIds, configuredIds, modelIds, labels };
}

function policyChanged(current: ModelPolicy, saved: ModelPolicy) {
  if (current.inheritParent !== saved.inheritParent) return true;
  if (current.inheritParent) return false;
  return current.ids.length !== saved.ids.length || current.ids.some((id) => !saved.ids.includes(id));
}

export function useWorkspaceModelPolicy(scope: WorkspaceModelPolicyScope) {
  const query = useFactoryLLMModels(scope.organizationId, scope.factoryId, scope.provider, scope.fundingSource, true);
  const update = useUpdateFactoryLLMModels(scope.organizationId, scope.factoryId);
  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState<ModelPolicy | null>(null);
  const [confirmed, setConfirmed] = useState<{ base: string; policy: ModelPolicy } | null>(null);
  const [feedback, setFeedback] = useState<{ message: string; error: boolean } | null>(null);
  const response = query.data ?? {};
  const catalog = modelCatalog(response);
  const serverPolicy = {
    inheritParent: response.inheritParent === true,
    ids: response.inheritParent ? catalog.availableIds : catalog.configuredIds,
  };
  const signature = JSON.stringify(serverPolicy);
  const saved = confirmed?.base === signature ? confirmed.policy : serverPolicy;
  const policy = draft ?? saved;
  const selected = policy.inheritParent ? catalog.availableIds : policy.ids;
  const edit = (next: ModelPolicy) => {
    setFeedback(null);
    setDraft(next);
  };
  const save = async () => {
    setFeedback(null);
    try {
      await update.mutateAsync({
        provider: scope.provider,
        fundingSource: scope.fundingSource,
        allowedModels: policy.inheritParent ? [] : policy.ids,
        inheritParent: policy.inheritParent,
      });
      setConfirmed({ base: signature, policy });
      setDraft(null);
      setFeedback({ message: COPY.saveSuccess, error: false });
    } catch (error) {
      setFeedback({ message: getApiErrorMessage(error, COPY.saveError), error: true });
    }
  };
  return {
    query,
    ...catalog,
    selected,
    policy,
    feedback,
    search,
    setSearch,
    save,
    saving: update.isPending,
    changed: policyChanged(policy, saved),
    setInheritance: (inheritParent: boolean) => edit({ inheritParent, ids: selected }),
    toggle: (id: string, checked: boolean) =>
      edit({ inheritParent: false, ids: checked ? [...selected, id] : selected.filter((value) => value !== id) }),
    toggleAll: () =>
      edit({
        inheritParent: false,
        ids: catalog.modelIds.every((id) => selected.includes(id)) ? [] : catalog.modelIds,
      }),
  };
}
