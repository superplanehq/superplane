import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input, InputGroup } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import { usePermissions } from "@/contexts/usePermissions";
import { BYOK_PROVIDERS, useBYOKLLMModels, useUpdateBYOKLLMModels } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { filterModelIds } from "@/lib/hostedLLMModels";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";

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
  const { data, isLoading } = useBYOKLLMModels(organizationId, provider, true);
  const update = useUpdateBYOKLLMModels(organizationId);
  const [query, setQuery] = useState("");
  const [draft, setDraft] = useState<string[] | null>(null);

  const selected = draft ?? (data?.selected ?? []).map((model) => model.id ?? "").filter(Boolean);
  const candidates = (data?.candidates ?? []).map((model) => model.id ?? "").filter(Boolean);
  const modelIds = candidates.length > 0 ? candidates : selected;
  const visibleModels = useMemo(() => filterModelIds(modelIds, query), [modelIds, query]);

  const save = async () => {
    try {
      await update.mutateAsync({ provider, allowedModels: selected });
      setDraft(null);
      showSuccessToast("Selected models saved.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Unable to save selected models."));
    }
  };

  return (
    <div>
      <p className="text-sm font-medium">{hostedProviderLabel(provider)}</p>
      {isLoading ? (
        <Text className="mt-2 text-sm text-gray-500 dark:text-gray-400">Loading models...</Text>
      ) : !data?.connected ? (
        <Text className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          Connect a {hostedProviderLabel(provider)} key on Integrations, then select models.
        </Text>
      ) : modelIds.length === 0 ? (
        <Text className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          List models from the connected key, then select models for this organization.
        </Text>
      ) : (
        <div className="mt-3 space-y-3">
          <InputGroup className="relative">
            <Search
              size={14}
              className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <Input
              type="search"
              className="pl-9"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search models..."
              aria-label={`Search ${hostedProviderLabel(provider)} models`}
            />
          </InputGroup>
          <Text className="text-xs text-gray-500 dark:text-gray-400">
            {selected.length} of {modelIds.length} models selected
          </Text>
          <div className="max-h-56 space-y-2 overflow-auto">
            {visibleModels.map((model) => (
              <label key={model} className="flex items-center gap-2 text-sm">
                <Checkbox
                  checked={selected.includes(model)}
                  disabled={!canUpdate || update.isPending}
                  onChange={(event) => {
                    const checked = event.currentTarget.checked;
                    setDraft(checked ? [...selected, model] : selected.filter((id) => id !== model));
                  }}
                />
                <span className="font-mono text-xs">{model}</span>
              </label>
            ))}
          </div>
          {canUpdate ? (
            <Button type="button" onClick={() => void save()} disabled={update.isPending || draft === null}>
              {update.isPending ? "Saving..." : "Select models"}
            </Button>
          ) : null}
        </div>
      )}
    </div>
  );
}
