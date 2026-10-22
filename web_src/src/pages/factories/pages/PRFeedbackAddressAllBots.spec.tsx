import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { useConnectedIntegrations, useIntegrationResources } from "@/hooks/useIntegrations";
import { unmockedSrc } from "@/test/unmockedModule";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";

vi.mock("@/hooks/useIntegrations", () => {
  const actual = unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations");
  return {
    ...actual,
    useConnectedIntegrations: vi.fn(() => ({ data: [] })),
    useIntegrationResources: vi.fn(() => ({
      data: [{ type: "review_bot", id: "coderabbitai", name: "coderabbitai[bot]" }],
      isLoading: false,
      isPending: false,
      isFetching: false,
      isError: false,
    })),
  };
});

function allBotsDraft(): PRFeedbackDraftSettings {
  return {
    source: "discussion",
    name: "Address PR feedback",
    repository: "acme/payments",
    mention: "@superplaneagent",
    ignoreBots: true,
    addressAllBots: true,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    runnerIntegrationIds: [],
  };
}

function renderSettings(onSave = vi.fn()) {
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PRFeedbackSettingsPopup
          organizationId="org-1"
          githubIntegrationId="gh-1"
          settings={allBotsDraft()}
          healthy
          onSave={onSave}
          onClose={vi.fn()}
          fixed={false}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { onSave };
}

describe("PR feedback all-bot comments", () => {
  beforeEach(() => {
    vi.mocked(useConnectedIntegrations).mockReturnValue({ data: [], isLoading: false, error: null } as never);
    vi.mocked(useIntegrationResources).mockReturnValue({
      data: [{ type: "review_bot", id: "coderabbitai", name: "coderabbitai[bot]" }],
      isLoading: false,
      isPending: false,
      isFetching: false,
      isError: false,
    } as never);
  });

  it("shows the all-bots choice and clears it when ignore is saved", async () => {
    const user = userEvent.setup();
    const { onSave } = renderSettings();

    expect(screen.getByRole("radio", { name: "All bots" })).toBeChecked();

    await user.click(screen.getByRole("radio", { name: "Ignore bots" }));
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ addressAllBots: false, allowedBots: [] }));
  });

  it("clears the all-bots choice when a selected bot list is saved", async () => {
    const user = userEvent.setup();
    const { onSave } = renderSettings();

    await user.click(screen.getByRole("radio", { name: "Choose bots" }));
    await user.click(screen.getByTestId("discussion-setup-bot-coderabbitai"));
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({ addressAllBots: false, allowedBots: ["coderabbitai"] }),
    );
  });
});
