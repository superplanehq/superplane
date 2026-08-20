import { Text } from "@/components/Text/text";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  bpsToPercentInput,
  centsToDollarInput,
  dollarInputToCents,
  hostedProviderLabel,
  percentInputToBps,
} from "@/lib/hostedCredit";
import { Switch } from "@/ui/switch";
import React, { useCallback, useEffect, useState } from "react";

type HostedLLMProvider = {
  provider: string;
  enabled: boolean;
  api_key_configured: boolean;
  base_url: string;
  allowed_models: string[];
};

type InstallationLLMSettings = {
  welcome_grant_cents: number;
  markup_bps: number;
  warning_threshold_bps: number;
  providers: HostedLLMProvider[];
};

type ProviderForm = {
  enabled: boolean;
  apiKey: string;
  baseURL: string;
  allowedModels: string[];
  listedModels: string[];
};

const emptyProviderForm = (provider?: HostedLLMProvider): ProviderForm => ({
  enabled: provider?.enabled ?? false,
  apiKey: "",
  baseURL: provider?.base_url ?? "",
  allowedModels: provider?.allowed_models ?? [],
  listedModels: provider?.allowed_models ?? [],
});

const getErrorMessage = async (response: Response, fallback: string) => {
  const text = await response.text();
  if (text.trim() === "") {
    return fallback;
  }

  return text;
};

export function HostedLLMSettings() {
  const [settings, setSettings] = useState<InstallationLLMSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [welcomeDollars, setWelcomeDollars] = useState("50.00");
  const [markupPercent, setMarkupPercent] = useState("20");
  const [warningPercent, setWarningPercent] = useState("20");
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [providers, setProviders] = useState<Record<string, ProviderForm>>({});
  const [savingProvider, setSavingProvider] = useState<string | null>(null);
  const [listingProvider, setListingProvider] = useState<string | null>(null);

  const applySettings = useCallback((data: InstallationLLMSettings) => {
    setSettings(data);
    setWelcomeDollars(centsToDollarInput(data.welcome_grant_cents));
    setMarkupPercent(bpsToPercentInput(data.markup_bps));
    setWarningPercent(bpsToPercentInput(data.warning_threshold_bps));
    setProviders((current) => {
      const next: Record<string, ProviderForm> = {};
      for (const provider of data.providers ?? []) {
        next[provider.provider] = {
          ...emptyProviderForm(provider),
          listedModels: current[provider.provider]?.listedModels?.length
            ? current[provider.provider].listedModels
            : provider.allowed_models,
        };
      }
      return next;
    });
  }, []);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch("/admin/api/installation/llm-settings", { credentials: "include" });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to load hosted LLM settings"));
      }
      applySettings(await response.json());
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load hosted LLM settings");
    } finally {
      setLoading(false);
    }
  }, [applySettings]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const savePolicy = async () => {
    setSavingPolicy(true);
    try {
      const response = await fetch("/admin/api/installation/llm-settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          welcome_grant_cents: dollarInputToCents(welcomeDollars),
          markup_bps: percentInputToBps(markupPercent),
          warning_threshold_bps: percentInputToBps(warningPercent),
        }),
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to update hosted LLM settings"));
      }
      applySettings(await response.json());
      showSuccessToast("Hosted LLM policy updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update hosted LLM settings");
    } finally {
      setSavingPolicy(false);
    }
  };

  const updateProviderForm = (provider: string, patch: Partial<ProviderForm>) => {
    setProviders((current) => ({
      ...current,
      [provider]: { ...current[provider], ...patch },
    }));
  };

  const listModels = async (provider: string) => {
    const form = providers[provider];
    setListingProvider(provider);
    try {
      const body: Record<string, unknown> = { base_url: form.baseURL.trim() };
      if (form.apiKey.trim() !== "") {
        body.api_key = form.apiKey.trim();
      }
      const response = await fetch(`/admin/api/installation/llm-providers/${provider}/models`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Unable to list models from the provider"));
      }
      const data: { models?: Array<{ id: string }> } = await response.json();
      const ids = (data.models ?? []).map((model) => model.id).filter((id) => id.trim() !== "");
      updateProviderForm(provider, { listedModels: ids.length > 0 ? ids : form.allowedModels });
      showSuccessToast("Model list updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Unable to list models from the provider");
    } finally {
      setListingProvider(null);
    }
  };

  const saveProvider = async (provider: string) => {
    const form = providers[provider];
    setSavingProvider(provider);
    try {
      const body: Record<string, unknown> = {
        enabled: form.enabled,
        base_url: form.baseURL.trim(),
        allowed_models: form.allowedModels,
      };
      if (form.apiKey.trim() !== "") {
        body.api_key = form.apiKey.trim();
      }
      const response = await fetch(`/admin/api/installation/llm-providers/${provider}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to update hosted LLM provider"));
      }
      applySettings(await response.json());
      showSuccessToast(`${hostedProviderLabel(provider)} hosted settings updated`);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update hosted LLM provider");
    } finally {
      setSavingProvider(null);
    }
  };

  const toggleAllowedModel = (provider: string, model: string, checked: boolean) => {
    const form = providers[provider];
    const next = checked
      ? Array.from(new Set([...form.allowedModels, model]))
      : form.allowedModels.filter((id) => id !== model);
    updateProviderForm(provider, { allowedModels: next });
  };

  const policyChanged =
    settings != null &&
    (dollarInputToCents(welcomeDollars) !== settings.welcome_grant_cents ||
      percentInputToBps(markupPercent) !== settings.markup_bps ||
      percentInputToBps(warningPercent) !== settings.warning_threshold_bps);

  return (
    <section className="border-t border-slate-200 py-6 dark:border-gray-700/70">
      <div className="max-w-2xl">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Hosted LLM</p>
        <h2 className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100">SuperPlane-hosted models</h2>
        <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Configure provider keys, model allowlists, welcome credit, and markup for SuperPlane-hosted runner
          credentials.
        </Text>
      </div>

      {loading && !settings ? (
        <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">Loading hosted LLM settings...</Text>
      ) : (
        <>
          <div className="mt-6 grid gap-4 md:grid-cols-3">
            <div>
              <Label className="mb-2 block text-left">Welcome grant (USD)</Label>
              <InputGroup>
                <Input
                  data-testid="installation-llm-welcome"
                  value={welcomeDollars}
                  onChange={(event) => setWelcomeDollars(event.target.value)}
                  placeholder="50.00"
                />
              </InputGroup>
              <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Granted once when an organization is created. Set to 0 to disable grants.
              </Text>
            </div>
            <div>
              <Label className="mb-2 block text-left">Markup percent</Label>
              <InputGroup>
                <Input
                  data-testid="installation-llm-markup"
                  value={markupPercent}
                  onChange={(event) => setMarkupPercent(event.target.value)}
                  placeholder="20"
                />
              </InputGroup>
              <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Applied to SuperPlane-hosted spend. Organization members cannot see this value.
              </Text>
            </div>
            <div>
              <Label className="mb-2 block text-left">Warning threshold percent</Label>
              <InputGroup>
                <Input
                  data-testid="installation-llm-warning"
                  value={warningPercent}
                  onChange={(event) => setWarningPercent(event.target.value)}
                  placeholder="20"
                />
              </InputGroup>
              <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Show a warning when remaining credit is at or below this percent of the grant total.
              </Text>
            </div>
          </div>

          <div className="mt-5">
            <Button
              type="button"
              data-testid="installation-llm-policy-save"
              onClick={savePolicy}
              disabled={savingPolicy || !policyChanged}
            >
              {savingPolicy ? "Saving..." : "Save hosted LLM policy"}
            </Button>
          </div>

          <div className="mt-8 space-y-6">
            {(settings?.providers ?? []).map((provider) => {
              const form = providers[provider.provider] ?? emptyProviderForm(provider);
              const modelChoices = Array.from(new Set([...form.listedModels, ...form.allowedModels]));
              return (
                <div key={provider.provider} className="rounded-md border border-slate-200 p-4 dark:border-gray-700/70">
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
                    <Switch
                      data-testid={`installation-llm-${provider.provider}-enabled`}
                      checked={form.enabled}
                      onCheckedChange={(checked) => updateProviderForm(provider.provider, { enabled: checked })}
                    />
                  </div>

                  <div className="mt-4 grid gap-4 md:grid-cols-2">
                    <div>
                      <Label className="mb-2 block text-left">API key</Label>
                      <InputGroup>
                        <Input
                          type="password"
                          className="ph-no-capture"
                          data-testid={`installation-llm-${provider.provider}-api-key`}
                          value={form.apiKey}
                          onChange={(event) => updateProviderForm(provider.provider, { apiKey: event.target.value })}
                          placeholder={
                            provider.api_key_configured ? "Leave blank to keep the current key" : "Provider API key"
                          }
                        />
                      </InputGroup>
                    </div>
                    <div>
                      <Label className="mb-2 block text-left">Base URL (optional)</Label>
                      <InputGroup>
                        <Input
                          value={form.baseURL}
                          onChange={(event) => updateProviderForm(provider.provider, { baseURL: event.target.value })}
                          placeholder="Use the provider default"
                        />
                      </InputGroup>
                    </div>
                  </div>

                  <div className="mt-4 flex flex-wrap items-center gap-3">
                    <Button
                      type="button"
                      variant="outline"
                      data-testid={`installation-llm-${provider.provider}-list-models`}
                      onClick={() => listModels(provider.provider)}
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
                    <div className="mt-4 max-h-48 space-y-2 overflow-auto">
                      {modelChoices.map((model) => (
                        <label key={model} className="flex items-center gap-2 text-sm text-gray-800 dark:text-gray-100">
                          <Checkbox
                            checked={form.allowedModels.includes(model)}
                            onChange={(event) =>
                              toggleAllowedModel(provider.provider, model, event.currentTarget.checked)
                            }
                          />
                          <span className="font-mono text-xs">{model}</span>
                        </label>
                      ))}
                    </div>
                  )}

                  <div className="mt-4">
                    <Button
                      type="button"
                      data-testid={`installation-llm-${provider.provider}-save`}
                      onClick={() => saveProvider(provider.provider)}
                      disabled={savingProvider === provider.provider}
                    >
                      {savingProvider === provider.provider
                        ? "Saving..."
                        : `Save ${hostedProviderLabel(provider.provider)}`}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}
    </section>
  );
}
