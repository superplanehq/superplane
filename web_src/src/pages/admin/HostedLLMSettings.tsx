import { Text } from "@/components/Text/text";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  bpsToPercentInput,
  centsToDollarInput,
  dollarInputToCents,
  hostedProviderLabel,
  parseDaysInput,
  percentInputToBps,
} from "@/lib/hostedCredit";
import { uniqueSortedModelIds } from "@/lib/hostedLLMModels";
import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from "react";
import {
  emptyProviderForm,
  fetchInstallationLLMSettings,
  patchHostedLLMProvider,
  patchInstallationLLMPolicy,
  postProviderModels,
  type InstallationLLMSettings,
  type ProviderForm,
} from "./hostedLLMSettingsApi";
import { HostedLLMDefaultModelField } from "./HostedLLMDefaultModelField";
import { HostedLLMPolicyFields } from "./HostedLLMPolicyFields";
import { HostedLLMProviderCard } from "./HostedLLMProviderCard";
import { defaultModelKeyFromSettings, hostedDefaultModelOptions, parseDefaultModelKey } from "./hostedLLMDefaultModel";

export function HostedLLMSettings() {
  const model = useHostedLLMSettings();

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

      {model.loading && !model.settings ? (
        <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">Loading hosted LLM settings...</Text>
      ) : (
        <>
          <HostedLLMPolicyFields
            welcomeDollars={model.welcomeDollars}
            welcomeTTLDays={model.welcomeTTLDays}
            markupPercent={model.markupPercent}
            warningPercent={model.warningPercent}
            savingPolicy={model.savingPolicy}
            policyChanged={model.policyChanged}
            policyValid={model.policyValid}
            onWelcomeChange={model.setWelcomeDollars}
            onWelcomeTTLChange={model.setWelcomeTTLDays}
            onMarkupChange={model.setMarkupPercent}
            onWarningChange={model.setWarningPercent}
            onSave={model.savePolicy}
          />
          <div className="mt-8 space-y-6">
            {(model.settings?.providers ?? []).map((provider) => (
              <HostedLLMProviderCard
                key={provider.provider}
                provider={provider}
                form={model.providers[provider.provider] ?? emptyProviderForm(provider)}
                listingProvider={model.listingProvider}
                savingProvider={model.savingProvider}
                onFormChange={model.updateProviderForm}
                onListModels={model.listModels}
                onToggleModel={model.toggleAllowedModel}
                onSave={model.saveProvider}
              />
            ))}
          </div>
          <HostedLLMDefaultModelField
            options={model.defaultModelOptions}
            value={model.defaultModelKey}
            saving={model.savingDefaultModel}
            changed={model.defaultModelChanged}
            onChange={model.setDefaultModelKey}
            onSave={model.saveDefaultModel}
          />
        </>
      )}
    </section>
  );
}

function useHostedLLMSettings() {
  const [settings, setSettings] = useState<InstallationLLMSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [welcomeDollars, setWelcomeDollars] = useState("50.00");
  const [welcomeTTLDays, setWelcomeTTLDays] = useState("14");
  const [markupPercent, setMarkupPercent] = useState("20");
  const [warningPercent, setWarningPercent] = useState("20");
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [providers, setProviders] = useState<Record<string, ProviderForm>>({});
  const [defaultModelKey, setDefaultModelKey] = useState("");
  const [savingDefaultModel, setSavingDefaultModel] = useState(false);

  const applySettings = useCallback((data: InstallationLLMSettings) => {
    setSettings(data);
    setWelcomeDollars(centsToDollarInput(data.welcome_grant_cents));
    setWelcomeTTLDays(String(data.welcome_grant_ttl_days ?? 14));
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
    setDefaultModelKey(defaultModelKeyFromSettings(data));
  }, []);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      applySettings(await fetchInstallationLLMSettings());
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
      applySettings(
        await patchInstallationLLMPolicy({
          welcome_grant_cents: dollarInputToCents(welcomeDollars),
          welcome_grant_ttl_days: parseDaysInput(welcomeTTLDays) ?? 0,
          markup_bps: percentInputToBps(markupPercent),
          warning_threshold_bps: percentInputToBps(warningPercent),
        }),
      );
      showSuccessToast("Hosted LLM policy updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update hosted LLM settings");
    } finally {
      setSavingPolicy(false);
    }
  };

  const saveDefaultModel = async () => {
    setSavingDefaultModel(true);
    try {
      const parsed = parseDefaultModelKey(defaultModelKey);
      applySettings(
        await patchInstallationLLMPolicy({
          default_hosted_provider: parsed.provider,
          default_hosted_model: parsed.model,
        }),
      );
      showSuccessToast("SuperPlane agent model updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update SuperPlane agent model");
    } finally {
      setSavingDefaultModel(false);
    }
  };

  const providerActions = useHostedLLMProviderActions(providers, setProviders, applySettings);
  const parsedWelcomeTTLDays = parseDaysInput(welcomeTTLDays);
  const policyChanged =
    settings != null &&
    (dollarInputToCents(welcomeDollars) !== settings.welcome_grant_cents ||
      (parsedWelcomeTTLDays ?? -1) !== settings.welcome_grant_ttl_days ||
      percentInputToBps(markupPercent) !== settings.markup_bps ||
      percentInputToBps(warningPercent) !== settings.warning_threshold_bps);
  const policyValid = parsedWelcomeTTLDays != null;
  const savedDefaultModelKey = defaultModelKeyFromSettings(settings);
  const defaultModelOptions = hostedDefaultModelOptions(settings?.providers ?? [], providers);
  const defaultModelChanged = defaultModelKey !== savedDefaultModelKey;

  return {
    settings,
    loading,
    welcomeDollars,
    setWelcomeDollars,
    welcomeTTLDays,
    setWelcomeTTLDays,
    markupPercent,
    setMarkupPercent,
    warningPercent,
    setWarningPercent,
    savingPolicy,
    providers,
    policyChanged,
    policyValid,
    savePolicy,
    defaultModelKey,
    setDefaultModelKey,
    defaultModelOptions,
    defaultModelChanged,
    savingDefaultModel,
    saveDefaultModel,
    ...providerActions,
  };
}

function useHostedLLMProviderActions(
  providers: Record<string, ProviderForm>,
  setProviders: Dispatch<SetStateAction<Record<string, ProviderForm>>>,
  applySettings: (data: InstallationLLMSettings) => void,
) {
  const [savingProvider, setSavingProvider] = useState<string | null>(null);
  const [listingProvider, setListingProvider] = useState<string | null>(null);

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
      const data = await postProviderModels(provider, body);
      const ids = uniqueSortedModelIds((data.models ?? []).map((model) => model.id));
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
      if (form.managementKey.trim() !== "") {
        body.management_key = form.managementKey.trim();
      }
      applySettings(await patchHostedLLMProvider(provider, body));
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

  return {
    savingProvider,
    listingProvider,
    updateProviderForm,
    listModels,
    toggleAllowedModel,
    saveProvider,
  };
}
