import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { HostedLLMSettings } from "./HostedLLMSettings";
import type { InstallationLLMSettings } from "./hostedLLMSettingsApi";

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

const settingsWithOpenRouterModels: InstallationLLMSettings = {
  welcome_grant_cents: 5000,
  markup_bps: 2000,
  warning_threshold_bps: 2000,
  providers: [
    {
      provider: "anthropic",
      enabled: false,
      api_key_configured: false,
      management_key_configured: false,
      base_url: "",
      allowed_models: [],
    },
    {
      provider: "openai",
      enabled: false,
      api_key_configured: false,
      management_key_configured: false,
      base_url: "",
      allowed_models: [],
    },
    {
      provider: "openrouter",
      enabled: true,
      api_key_configured: true,
      management_key_configured: true,
      base_url: "",
      allowed_models: ["openai/gpt-4.1", "anthropic/claude-sonnet-4"],
    },
  ],
};

const mockSettingsFetch = (settings = settingsWithOpenRouterModels) => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      return new Response(JSON.stringify(settings), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("HostedLLMSettings", () => {
  it("sorts listed models by name and filters them with search", async () => {
    mockSettingsFetch();
    const user = userEvent.setup();

    render(<HostedLLMSettings />);

    const list = await screen.findByTestId("installation-llm-openrouter-model-list");
    expect(
      within(list)
        .getAllByText(/anthropic\/|openai\//)
        .map((node) => node.textContent),
    ).toEqual(["anthropic/claude-sonnet-4", "openai/gpt-4.1"]);

    const search = screen.getByLabelText("Search models");
    expect(search).toHaveAttribute("id", "installation-llm-openrouter-model-search");
    await user.type(search, "gpt");

    expect(within(list).queryByText("anthropic/claude-sonnet-4")).not.toBeInTheDocument();
    expect(within(list).getByText("openai/gpt-4.1")).toBeInTheDocument();
  });

  it("explains when no models match the search", async () => {
    mockSettingsFetch();
    const user = userEvent.setup();

    render(<HostedLLMSettings />);

    await screen.findByTestId("installation-llm-openrouter-model-list");
    await user.type(screen.getByTestId("installation-llm-openrouter-model-search"), "does-not-exist");

    expect(screen.getByText("No models match this search.")).toBeInTheDocument();
  });

  it("lists SuperPlane agent models from a saved OpenRouter allowlist when the switch is off", async () => {
    mockSettingsFetch({
      ...settingsWithOpenRouterModels,
      providers: settingsWithOpenRouterModels.providers.map((provider) =>
        provider.provider === "openrouter" ? { ...provider, enabled: false } : provider,
      ),
    });

    render(<HostedLLMSettings />);

    const trigger = await screen.findByTestId("installation-llm-default-model");
    expect(trigger).toHaveTextContent("No SuperPlane agent model");
    await userEvent.click(trigger);
    expect(await screen.findByRole("option", { name: "OpenRouter - openai/gpt-4.1" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "OpenRouter - anthropic/claude-sonnet-4" })).toBeInTheDocument();
  });

  it("lists SuperPlane agent models as provider - model", async () => {
    mockSettingsFetch({
      ...settingsWithOpenRouterModels,
      default_hosted_provider: "openrouter",
      default_hosted_model: "openai/gpt-4.1",
    });

    render(<HostedLLMSettings />);

    const trigger = await screen.findByTestId("installation-llm-default-model");
    expect(trigger).toHaveTextContent("OpenRouter - openai/gpt-4.1");
    await userEvent.click(trigger);
    expect(await screen.findByRole("option", { name: "OpenRouter - openai/gpt-4.1" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "OpenRouter - anthropic/claude-sonnet-4" })).toBeInTheDocument();
  });

  it("saves the selected SuperPlane agent model", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      const method = init?.method ?? "GET";
      const body =
        method === "PATCH"
          ? {
              ...settingsWithOpenRouterModels,
              default_hosted_provider: "openrouter",
              default_hosted_model: "anthropic/claude-sonnet-4",
            }
          : {
              ...settingsWithOpenRouterModels,
              default_hosted_provider: "openrouter",
              default_hosted_model: "openai/gpt-4.1",
            };
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<HostedLLMSettings />);

    const trigger = await screen.findByTestId("installation-llm-default-model");
    await user.click(trigger);
    await user.click(await screen.findByRole("option", { name: "OpenRouter - anthropic/claude-sonnet-4" }));
    await user.click(screen.getByTestId("installation-llm-default-model-save"));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/admin/api/installation/llm-settings",
        expect.objectContaining({
          method: "PATCH",
          body: JSON.stringify({
            default_hosted_provider: "openrouter",
            default_hosted_model: "anthropic/claude-sonnet-4",
          }),
        }),
      );
    });
  });

  it("shows the OpenRouter provisioning API key and hides it on other providers", async () => {
    mockSettingsFetch();

    render(<HostedLLMSettings />);

    const provisioningKey = await screen.findByLabelText("Provisioning API Key");
    expect(provisioningKey).toHaveAttribute("id", "installation-llm-openrouter-management-key");
    expect(provisioningKey).toHaveAttribute("placeholder", "Leave blank to keep the current key");
    expect(screen.getByLabelText("Enable OpenRouter")).toHaveAttribute("id", "installation-llm-openrouter-enabled");
    const apiKeys = screen.getAllByLabelText("API key");
    expect(apiKeys.map((node) => node.getAttribute("id"))).toEqual([
      "installation-llm-anthropic-api-key",
      "installation-llm-openai-api-key",
      "installation-llm-openrouter-api-key",
    ]);
    expect(screen.getAllByLabelText("Base URL (optional)")[2]).toHaveAttribute(
      "id",
      "installation-llm-openrouter-base-url",
    );
    expect(screen.getByText(/uses this key to create a short-lived OpenRouter key/)).toBeInTheDocument();
    expect(screen.getByText(/does not send this key to the runner/)).toBeInTheDocument();
    expect(screen.queryByTestId("installation-llm-anthropic-management-key")).not.toBeInTheDocument();
    expect(screen.queryByTestId("installation-llm-openai-management-key")).not.toBeInTheDocument();
  });

  it("saves a typed OpenRouter provisioning API key", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => {
      return new Response(JSON.stringify(settingsWithOpenRouterModels), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<HostedLLMSettings />);

    await user.type(await screen.findByTestId("installation-llm-openrouter-management-key"), "sk-or-mgmt");
    await user.click(screen.getByTestId("installation-llm-openrouter-save"));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/admin/api/installation/llm-providers/openrouter",
        expect.objectContaining({
          method: "PATCH",
          body: JSON.stringify({
            enabled: true,
            base_url: "",
            allowed_models: ["openai/gpt-4.1", "anthropic/claude-sonnet-4"],
            management_key: "sk-or-mgmt",
          }),
        }),
      );
    });
  });
});
