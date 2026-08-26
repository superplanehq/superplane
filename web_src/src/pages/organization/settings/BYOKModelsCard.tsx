import { Button } from "@/components/ui/button";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { BYOK_PROVIDERS, useBYOKLLMModels, useUpdateBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useState } from "react";
import { Link } from "react-router";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { ModelAllowlistEditor } from "./ModelAllowlistEditor";

const PROVIDER_LIST = [...BYOK_PROVIDERS];

export function BYOKModelsCard({ organizationId }: { organizationId: string }) {
  const { canAct } = usePermissions();
  const canUpdate = canAct("org", "update");
  const integrationsHref = useIntegrationsBasePath(organizationId);

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <p className="text-sm font-semibold">Use your keys</p>
      <Text className="mt-1 text-sm text-muted-foreground">
        Connect Anthropic, OpenAI, or OpenRouter on Integrations. Then select models for this organization.
      </Text>
      <Link to={integrationsHref} className="mt-2 inline-block text-sm text-foreground underline underline-offset-2">
        Open Integrations
      </Link>
      <div className="mt-4 space-y-6">
        {PROVIDER_LIST.map((provider) => (
          <BYOKProviderAllowlist
            key={provider}
            organizationId={organizationId}
            provider={provider}
            canUpdate={canUpdate}
          />
        ))}
      </div>
    </div>
  );
}

function BYOKProviderAllowlist({
  organizationId,
  provider,
  canUpdate,
}: {
  organizationId: string;
  provider: string;
  canUpdate: boolean;
}) {
  const { data, isLoading, error } = useBYOKLLMModels(organizationId, provider, true);
  const update = useUpdateBYOKLLMModels(organizationId);
  const [query, setQuery] = useState("");
  const [draft, setDraft] = useState<string[] | null>(null);
  const selected = draft ?? (data?.selected ?? []).map((model) => model.id ?? "").filter(Boolean);
  const candidates = (data?.candidates ?? []).map((model) => model.id ?? "").filter(Boolean);
  const modelIds = candidates.length > 0 ? candidates : selected;

  const save = async () => {
    try {
      await update.mutateAsync({ provider, allowedModels: selected });
      setDraft(null);
      showSuccessToast("Selected models saved.");
    } catch (saveError) {
      showErrorToast(getApiErrorMessage(saveError, "Unable to save selected models."));
    }
  };

  return (
    <div>
      <p className="text-sm font-medium">{hostedProviderLabel(provider)}</p>
      <BYOKAllowlistBody
        canUpdate={canUpdate}
        connected={Boolean(data?.connected)}
        error={Boolean(error)}
        isLoading={isLoading}
        isPending={update.isPending}
        modelIds={modelIds}
        provider={provider}
        query={query}
        selected={selected}
        showSave={draft !== null}
        onQueryChange={setQuery}
        onSave={() => void save()}
        onToggle={(model, checked) => setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model))}
      />
    </div>
  );
}

function BYOKAllowlistBody({
  canUpdate,
  connected,
  error,
  isLoading,
  isPending,
  modelIds,
  provider,
  query,
  selected,
  showSave,
  onQueryChange,
  onSave,
  onToggle,
}: {
  canUpdate: boolean;
  connected: boolean;
  error: boolean;
  isLoading: boolean;
  isPending: boolean;
  modelIds: string[];
  provider: string;
  query: string;
  selected: string[];
  showSave: boolean;
  onQueryChange: (query: string) => void;
  onSave: () => void;
  onToggle: (model: string, checked: boolean) => void;
}) {
  const message = byokAllowlistMessage({ isLoading, error, connected, hasModels: modelIds.length > 0, provider });
  if (message) {
    return <Text className={`mt-2 text-sm ${error ? "text-destructive" : "text-muted-foreground"}`}>{message}</Text>;
  }

  return (
    <div className="mt-3 space-y-3">
      <ModelAllowlistEditor
        modelIds={modelIds}
        selected={selected}
        query={query}
        onQueryChange={onQueryChange}
        onToggle={onToggle}
        disabled={!canUpdate || isPending}
        searchLabel={`Search ${hostedProviderLabel(provider)} models`}
        showCount
      />
      {canUpdate ? (
        <Button type="button" onClick={onSave} disabled={isPending || !showSave}>
          {isPending ? "Saving..." : "Select models"}
        </Button>
      ) : null}
    </div>
  );
}

function byokAllowlistMessage({
  isLoading,
  error,
  connected,
  hasModels,
  provider,
}: {
  isLoading: boolean;
  error: boolean;
  connected: boolean;
  hasModels: boolean;
  provider: string;
}) {
  if (isLoading) {
    return "Loading models...";
  }
  if (error) {
    return "Unable to list models from the connected key.";
  }
  if (!connected) {
    return `Connect a ${hostedProviderLabel(provider)} key on Integrations, then select models.`;
  }
  if (!hasModels) {
    return "List models from the connected key, then select models for this organization.";
  }
  return null;
}
