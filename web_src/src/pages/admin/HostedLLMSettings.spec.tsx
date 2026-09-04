import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { HostedLLMSettings } from "./HostedLLMSettings";
import type { InstallationLLMSettings } from "./hostedLLMSettingsApi";

const settingsWithOpenRouterModels: InstallationLLMSettings = {
  welcome_grant_cents: 5000,
  markup_bps: 2000,
  warning_threshold_bps: 2000,
  providers: [
    { provider: "anthropic", enabled: false, api_key_configured: false, base_url: "", allowed_models: [] },
    { provider: "openai", enabled: false, api_key_configured: false, base_url: "", allowed_models: [] },
    {
      provider: "openrouter",
      enabled: true,
      api_key_configured: true,
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

    const search = screen.getByTestId("installation-llm-openrouter-model-search");
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

  it("lists SuperPlane agent models as provider - model", async () => {
    mockSettingsFetch({
      ...settingsWithOpenRouterModels,
      default_hosted_provider: "openrouter",
      default_hosted_model: "openai/gpt-4.1",
    });

    render(<HostedLLMSettings />);

    const select = await screen.findByTestId("installation-llm-default-model");
    expect(select).toHaveValue("openrouter::openai/gpt-4.1");
    expect(screen.getByRole("option", { name: "OpenRouter - openai/gpt-4.1" })).toBeInTheDocument();
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

    const select = await screen.findByTestId("installation-llm-default-model");
    await user.selectOptions(select, "openrouter::anthropic/claude-sonnet-4");
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
});
