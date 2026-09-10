import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { InputGroup } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { hostedProviderLabel } from "@/lib/hostedCredit";
import { filterModelIds, uniqueSortedModelIds } from "@/lib/hostedLLMModels";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";
import type { HostedLLMProvider, ProviderForm } from "./hostedLLMSettingsApi";

function providerFieldId(provider: string, field: string) {
  return `installation-llm-${provider}-${field}`;
}

export function HostedLLMProviderCard({
  provider,
  form,
  listingProvider,
  savingProvider,
  onFormChange,
  onListModels,
  onToggleModel,
  onSave,
}: {
  provider: HostedLLMProvider;
  form: ProviderForm;
  listingProvider: string | null;
  savingProvider: string | null;
  onFormChange: (provider: string, patch: Partial<ProviderForm>) => void;
  onListModels: (provider: string) => void;
  onToggleModel: (provider: string, model: string, checked: boolean) => void;
  onSave: (provider: string) => void;
}) {
  const modelChoices = uniqueSortedModelIds([...form.listedModels, ...form.allowedModels]);
  const enabledId = providerFieldId(provider.provider, "enabled");
  const apiKeyId = providerFieldId(provider.provider, "api-key");
  const baseUrlId = providerFieldId(provider.provider, "base-url");
  return (
    <div className="rounded-md border border-slate-200 p-4 dark:border-gray-700/70">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
            {hostedProviderLabel(provider.provider)}
          </h3>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {provider.api_key_configured
              ? "An API key is stored for this provider."
              : "No API key is stored for this provider."}
          </Text>
        </div>
        <div className="flex items-center">
          <Label htmlFor={enabledId} className="sr-only">
            Enable {hostedProviderLabel(provider.provider)}
          </Label>
          <Switch
            id={enabledId}
            data-testid={enabledId}
            checked={form.enabled}
            onCheckedChange={(checked) => onFormChange(provider.provider, { enabled: checked })}
          />
        </div>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <div>
          <Label htmlFor={apiKeyId} className="mb-2 block text-left">
            API key
          </Label>
          <InputGroup>
            <Input
              id={apiKeyId}
              type="password"
              className="ph-no-capture"
              data-testid={apiKeyId}
              value={form.apiKey}
              onChange={(event) => onFormChange(provider.provider, { apiKey: event.target.value })}
              placeholder={provider.api_key_configured ? "Leave blank to keep the current key" : "Provider API key"}
            />
          </InputGroup>
        </div>
        <div>
          <Label htmlFor={baseUrlId} className="mb-2 block text-left">
            Base URL (optional)
          </Label>
          <InputGroup>
            <Input
              id={baseUrlId}
              value={form.baseURL}
              onChange={(event) => onFormChange(provider.provider, { baseURL: event.target.value })}
              placeholder="Use the provider default"
            />
          </InputGroup>
        </div>
      </div>

      {provider.provider === "openrouter" && (
        <OpenRouterProvisioningKeyField provider={provider} form={form} onFormChange={onFormChange} />
      )}

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <Button
          type="button"
          variant="outline"
          data-testid={`installation-llm-${provider.provider}-list-models`}
          onClick={() => onListModels(provider.provider)}
          disabled={listingProvider === provider.provider}
        >
          {listingProvider === provider.provider ? "Listing models..." : "List models"}
        </Button>
        <Text className="text-xs text-gray-500 dark:text-gray-400">
          Uses the typed key, or the stored key if the field is blank.
        </Text>
      </div>

      {modelChoices.length === 0 ? (
        <Text className="mt-4 text-sm text-gray-500 dark:text-gray-400">
          List models, then select the allowlist for SuperPlane-hosted nodes.
        </Text>
      ) : (
        <HostedProviderModelAllowlist
          provider={provider.provider}
          modelIds={modelChoices}
          allowedModels={form.allowedModels}
          onToggle={(model, checked) => onToggleModel(provider.provider, model, checked)}
        />
      )}

      <div className="mt-4">
        <Button
          type="button"
          data-testid={`installation-llm-${provider.provider}-save`}
          onClick={() => onSave(provider.provider)}
          disabled={savingProvider === provider.provider}
        >
          {savingProvider === provider.provider ? "Saving..." : `Save ${hostedProviderLabel(provider.provider)}`}
        </Button>
      </div>
    </div>
  );
}

function OpenRouterProvisioningKeyField({
  provider,
  form,
  onFormChange,
}: {
  provider: HostedLLMProvider;
  form: ProviderForm;
  onFormChange: (provider: string, patch: Partial<ProviderForm>) => void;
}) {
  const managementKeyId = providerFieldId(provider.provider, "management-key");
  return (
    <div className="mt-4">
      <Label htmlFor={managementKeyId} className="mb-2 block text-left">
        Provisioning API Key
      </Label>
      <InputGroup>
        <Input
          id={managementKeyId}
          type="password"
          className="ph-no-capture"
          data-testid={managementKeyId}
          value={form.managementKey}
          onChange={(event) => onFormChange(provider.provider, { managementKey: event.target.value })}
          placeholder={
            provider.management_key_configured
              ? "Leave blank to keep the current key"
              : "OpenRouter provisioning API key"
          }
        />
      </InputGroup>
      <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        SuperPlane uses this key to create a short-lived OpenRouter key for each runner task. SuperPlane does not send
        this key to the runner.
      </Text>
    </div>
  );
}

function HostedProviderModelAllowlist({
  provider,
  modelIds,
  allowedModels,
  onToggle,
}: {
  provider: string;
  modelIds: string[];
  allowedModels: string[];
  onToggle: (model: string, checked: boolean) => void;
}) {
  const [query, setQuery] = useState("");
  const visibleModels = useMemo(() => filterModelIds(modelIds, query), [modelIds, query]);
  const selectedCount = allowedModels.length;
  const searchId = providerFieldId(provider, "model-search");

  return (
    <div className="mt-4 space-y-3">
      <Label htmlFor={searchId} className="sr-only">
        Search models
      </Label>
      <InputGroup className="relative">
        <Search
          size={14}
          className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
        />
        <Input
          id={searchId}
          type="search"
          className="pl-9"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search models..."
          data-testid={searchId}
        />
      </InputGroup>
      <Text className="text-xs text-gray-500 dark:text-gray-400">
        {selectedCount} of {modelIds.length} models selected
      </Text>
      {visibleModels.length === 0 ? (
        <Text className="text-sm text-gray-500 dark:text-gray-400">No models match this search.</Text>
      ) : (
        <div className="max-h-72 space-y-2 overflow-auto" data-testid={`installation-llm-${provider}-model-list`}>
          {visibleModels.map((model) => (
            <label key={model} className="flex items-center gap-2 text-sm text-gray-800 dark:text-gray-100">
              <Checkbox
                checked={allowedModels.includes(model)}
                onChange={(event) => onToggle(model, event.currentTarget.checked)}
              />
              <span className="font-mono text-xs">{model}</span>
            </label>
          ))}
        </div>
      )}
    </div>
  );
}
