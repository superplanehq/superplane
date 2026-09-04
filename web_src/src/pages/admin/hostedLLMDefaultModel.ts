import { hostedProviderLabel } from "@/lib/hostedCredit";

import type { HostedLLMProvider, InstallationLLMSettings, ProviderForm } from "./hostedLLMSettingsApi";

export function hostedDefaultModelOptions(
  providers: HostedLLMProvider[],
  forms: Record<string, ProviderForm> = {},
): Array<{ value: string; label: string }> {
  const options: Array<{ value: string; label: string }> = [];
  for (const provider of providers) {
    const form = forms[provider.provider];
    if (!providerOffersDefaultModels(provider, form)) continue;
    for (const model of form?.allowedModels ?? provider.allowed_models ?? []) {
      const trimmed = model.trim();
      if (trimmed === "") continue;
      options.push({
        value: `${provider.provider}::${trimmed}`,
        label: `${hostedProviderLabel(provider.provider)} - ${trimmed}`,
      });
    }
  }
  return options;
}

function providerOffersDefaultModels(provider: HostedLLMProvider, form: ProviderForm | undefined): boolean {
  const hasKey = provider.api_key_configured || Boolean(form?.apiKey.trim());
  if (!hasKey) return false;
  const allowed = form?.allowedModels ?? provider.allowed_models ?? [];
  return allowed.some((model) => model.trim() !== "");
}

export function defaultModelKeyFromSettings(settings: InstallationLLMSettings | null): string {
  const provider = settings?.default_hosted_provider?.trim() ?? "";
  const model = settings?.default_hosted_model?.trim() ?? "";
  if (provider === "" || model === "") {
    return "";
  }
  return `${provider}::${model}`;
}

export function parseDefaultModelKey(value: string): { provider: string; model: string } {
  const separator = value.indexOf("::");
  if (separator < 0) {
    return { provider: "", model: "" };
  }
  return { provider: value.slice(0, separator), model: value.slice(separator + 2) };
}
