import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useConnectedIntegrations } from "@/hooks/useIntegrations";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: vi.fn(() => ({ data: [] })),
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: ({ integrationName }: { integrationName?: string }) => (
    <span data-testid={`integration-icon-${integrationName ?? "unknown"}`} />
  ),
}));

function checksDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "checks",
    name: "Fix pull request checks",
    repository: "acme/app",
    mention: "",
    ignoreBots: false,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    runnerIntegrationIds: [],
    ...overrides,
  };
}

beforeEach(() => {
  vi.mocked(useConnectedIntegrations).mockReturnValue({
    data: [],
  } as ReturnType<typeof useConnectedIntegrations>);
});

function renderChecksPopup(
  onSave = vi.fn(),
  settings: PRFeedbackDraftSettings = checksDraft(),
  organizationId?: string,
) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <PRFeedbackSettingsPopup
        organizationId={organizationId}
        settings={settings}
        healthy
        onSave={onSave}
        onClose={vi.fn()}
        fixed={false}
      />
    </QueryClientProvider>,
  );
  return { onSave };
}

describe("PRFeedbackSettingsPopup check names", () => {
  it("adds a check name that contains a comma as one value", async () => {
    const user = userEvent.setup();
    renderChecksPopup();

    const input = screen.getByTestId("pr-feedback-check-names");
    await user.type(input, "lint, typecheck");
    await user.keyboard("{Enter}");

    const names = screen.getByTestId("pr-feedback-check-names-list");
    expect(within(names).getAllByRole("listitem")).toHaveLength(1);
    expect(names).toHaveTextContent("lint, typecheck");
    expect(input).toHaveValue("");
  });

  it("adds a second name with Add and keeps the first comma-containing name", async () => {
    const user = userEvent.setup();
    renderChecksPopup();

    await user.type(screen.getByTestId("pr-feedback-check-names"), "lint, typecheck");
    await user.keyboard("{Enter}");
    await user.type(screen.getByTestId("pr-feedback-check-names"), "unit");
    await user.click(screen.getByTestId("pr-feedback-check-names-add"));

    const names = screen.getByTestId("pr-feedback-check-names-list");
    expect(within(names).getAllByRole("listitem")).toHaveLength(2);
    expect(names).toHaveTextContent("lint, typecheck");
    expect(names).toHaveTextContent("unit");
  });

  it("includes a pending name that contains a comma when the user saves", async () => {
    const user = userEvent.setup();
    const { onSave } = renderChecksPopup();

    await user.type(screen.getByTestId("pr-feedback-check-names"), "lint, typecheck");
    await user.click(screen.getByTestId("pr-feedback-settings-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        checkNames: ["lint, typecheck"],
      }),
    );
  });

  it("removes a selected check name", async () => {
    const user = userEvent.setup();
    renderChecksPopup(vi.fn(), checksDraft({ checkNames: ["lint, typecheck", "unit"] }));

    await user.click(screen.getByRole("button", { name: "Remove check lint, typecheck" }));

    const names = screen.getByTestId("pr-feedback-check-names-list");
    expect(within(names).getAllByRole("listitem")).toHaveLength(1);
    expect(names).toHaveTextContent("unit");
    expect(names).not.toHaveTextContent("lint, typecheck");
  });
});

describe("PRFeedbackSettingsPopup additional integrations", () => {
  beforeEach(() => {
    vi.mocked(useConnectedIntegrations).mockReturnValue({
      data: [
        {
          metadata: { id: "int-circleci", name: "circleci-prod", integrationName: "circleci" },
          status: { state: "ready" },
        },
      ],
    } as ReturnType<typeof useConnectedIntegrations>);
  });

  it("shows the integration icon next to the integration name", () => {
    renderChecksPopup(vi.fn(), checksDraft(), "org-1");

    const row = screen.getByTestId("pr-feedback-integrations");
    expect(within(row).getByTestId("integration-icon-circleci")).toBeInTheDocument();
    expect(row).toHaveTextContent("circleci-prod");
    expect(within(row).getByRole("listitem").className).toContain("items-center");
  });
});
