export type HostedLLMProvider = {
  provider: string;
  enabled: boolean;
  api_key_configured: boolean;
  base_url: string;
  allowed_models: string[];
};

export type InstallationLLMSettings = {
  welcome_grant_cents: number;
  markup_bps: number;
  warning_threshold_bps: number;
  providers: HostedLLMProvider[];
};

export type ProviderForm = {
  enabled: boolean;
  apiKey: string;
  baseURL: string;
  allowedModels: string[];
  listedModels: string[];
};

const SETTINGS_PATH = "/admin/api/installation/llm-settings";

export const emptyProviderForm = (provider?: HostedLLMProvider): ProviderForm => ({
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

const parseJson = async (response: Response, fallback: string) => {
  if (!response.ok) {
    throw new Error(await getErrorMessage(response, fallback));
  }
  return response.json();
};

export const fetchInstallationLLMSettings = async (): Promise<InstallationLLMSettings> => {
  const response = await fetch(SETTINGS_PATH, { credentials: "include" });
  return parseJson(response, "Failed to load hosted LLM settings");
};

export const patchInstallationLLMPolicy = async (body: {
  welcome_grant_cents: number;
  markup_bps: number;
  warning_threshold_bps: number;
}): Promise<InstallationLLMSettings> => {
  const response = await fetch(SETTINGS_PATH, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(body),
  });
  return parseJson(response, "Failed to update hosted LLM settings");
};

export const postProviderModels = async (
  provider: string,
  body: Record<string, unknown>,
): Promise<{ models?: Array<{ id: string }> }> => {
  const response = await fetch(`/admin/api/installation/llm-providers/${provider}/models`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(body),
  });
  return parseJson(response, "Unable to list models from the provider");
};

export const patchHostedLLMProvider = async (
  provider: string,
  body: Record<string, unknown>,
): Promise<InstallationLLMSettings> => {
  const response = await fetch(`/admin/api/installation/llm-providers/${provider}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(body),
  });
  return parseJson(response, "Failed to update hosted LLM provider");
};
