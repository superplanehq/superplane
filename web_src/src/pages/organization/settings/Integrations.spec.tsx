import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
} from "../../../hooks/useIntegrations";
import { Integrations } from "./Integrations";

vi.mock("../../../hooks/useIntegrations", () => ({
  useAvailableIntegrations: vi.fn(),
  useConnectedIntegrations: vi.fn(),
  useCreateIntegration: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/posthog", () => ({
  isPostHogEnabled: false,
  posthog: { getSurveys: vi.fn() },
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: ({ integrationName }: { integrationName?: string }) => (
    <span data-testid={`icon-${integrationName ?? "unknown"}`} />
  ),
}));

vi.mock("@/lib/analytics", () => ({
  analytics: { integrationConnectStart: vi.fn(), integrationRequested: vi.fn() },
}));

function definition(name: string, label: string): IntegrationsIntegrationDefinition {
  return { name, label, description: `${label} description`, configuration: [] };
}

function instance(id: string, provider: string, name: string): OrganizationsIntegration {
  return {
    metadata: { id, name, integrationName: provider },
    status: { state: "ready" },
  };
}

const AVAILABLE = [
  definition("github", "GitHub"),
  definition("gitlab", "GitLab"),
  definition("slack", "Slack"),
  definition("datadog", "Datadog"),
];

function renderPage(connected: OrganizationsIntegration[] = []) {
  vi.mocked(useAvailableIntegrations).mockReturnValue({
    data: AVAILABLE,
    isLoading: false,
  } as unknown as ReturnType<typeof useAvailableIntegrations>);
  vi.mocked(useConnectedIntegrations).mockReturnValue({
    data: connected,
    isLoading: false,
  } as unknown as ReturnType<typeof useConnectedIntegrations>);
  vi.mocked(useCreateIntegration).mockReturnValue({
    mutateAsync: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
    reset: vi.fn(),
  } as unknown as ReturnType<typeof useCreateIntegration>);

  return render(
    <MemoryRouter>
      <Integrations organizationId="org-1" />
    </MemoryRouter>,
  );
}

function activeRowName() {
  const active = document.querySelector('[data-integration-row][aria-current="true"]');
  return active?.getAttribute("data-integration-row");
}

describe("Integrations settings page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("connected / available split", () => {
    it("separates connected providers from the rest, with counts", () => {
      renderPage([instance("i1", "github", "my-github")]);

      const connectedSection = screen.getByRole("region", { name: /connected/i });
      const availableSection = screen.getByRole("region", { name: /available/i });

      expect(within(connectedSection).getByText("GitHub")).toBeInTheDocument();
      expect(within(connectedSection).queryByText("Slack")).not.toBeInTheDocument();
      expect(within(availableSection).getByText("Slack")).toBeInTheDocument();

      expect(screen.getByRole("region", { name: /connected \(1\)/i })).toBeInTheDocument();
      expect(screen.getByRole("region", { name: /available \(3\)/i })).toBeInTheDocument();
    });

    it("hides the connected section when nothing is connected", () => {
      renderPage();

      expect(screen.queryByRole("region", { name: /connected/i })).not.toBeInTheDocument();
      expect(screen.getByRole("region", { name: /available \(4\)/i })).toBeInTheDocument();
    });
  });

  describe("filtering", () => {
    it("narrows the list and announces how many match", async () => {
      const user = userEvent.setup();
      renderPage();

      await user.type(screen.getByPlaceholderText(/filter integrations/i), "git");

      expect(screen.getByText("GitHub")).toBeInTheDocument();
      expect(screen.getByText("GitLab")).toBeInTheDocument();
      expect(screen.queryByText("Slack")).not.toBeInTheDocument();
      expect(screen.getByRole("status")).toHaveTextContent("2 of 4 integrations");
    });

    it("clears the query on Escape", async () => {
      const user = userEvent.setup();
      renderPage();

      const input = screen.getByPlaceholderText(/filter integrations/i);
      await user.type(input, "git");
      expect(input).toHaveValue("git");

      await user.keyboard("{Escape}");
      expect(input).toHaveValue("");
      expect(screen.getByText("Slack")).toBeInTheDocument();
    });
  });

  describe("keyboard navigation", () => {
    it("focuses the filter input when pressing /", async () => {
      const user = userEvent.setup();
      renderPage();

      const input = screen.getByPlaceholderText(/filter integrations/i);
      input.blur();
      expect(input).not.toHaveFocus();

      await user.keyboard("/");

      expect(input).toHaveFocus();
      expect(input).toHaveValue("");
    });

    it("shows no highlight until the keyboard is used", () => {
      renderPage([instance("i1", "github", "my-github")]);

      screen.getByPlaceholderText(/filter integrations/i).focus();

      expect(activeRowName()).toBeUndefined();
    });

    it("does not highlight rows on hover", async () => {
      const user = userEvent.setup();
      renderPage();

      await user.hover(screen.getByText("Slack"));

      expect(activeRowName()).toBeUndefined();
    });

    it("moves the highlight with the arrow keys", async () => {
      const user = userEvent.setup();
      renderPage([instance("i1", "github", "my-github")]);

      const input = screen.getByPlaceholderText(/filter integrations/i);
      input.focus();

      // The first press reveals the highlight in place rather than skipping a row.
      await user.keyboard("{ArrowDown}");
      expect(activeRowName()).toBe("github");

      await user.keyboard("{ArrowDown}");
      expect(activeRowName()).toBe("datadog");

      await user.keyboard("{ArrowUp}");
      expect(activeRowName()).toBe("github");
    });

    it("hides the highlight again when the filter is cleared", async () => {
      const user = userEvent.setup();
      renderPage();

      const input = screen.getByPlaceholderText(/filter integrations/i);
      await user.type(input, "git");
      expect(activeRowName()).toBe("github");

      await user.keyboard("{Escape}");
      expect(activeRowName()).toBeUndefined();
    });

    it("does not connect anything when Enter is pressed with no visible highlight", async () => {
      const user = userEvent.setup();
      renderPage();

      screen.getByPlaceholderText(/filter integrations/i).focus();
      await user.keyboard("{Enter}");

      expect(screen.queryByRole("heading", { name: /^connect /i })).not.toBeInTheDocument();
    });

    it("connects the highlighted integration on Enter", async () => {
      const user = userEvent.setup();
      renderPage();

      const input = screen.getByPlaceholderText(/filter integrations/i);
      input.focus();
      await user.keyboard("slack");

      expect(activeRowName()).toBe("slack");

      await user.keyboard("{Enter}");

      expect(screen.getByRole("heading", { name: /connect slack/i })).toBeInTheDocument();
    });

    it("highlights the first match again after the query changes", async () => {
      const user = userEvent.setup();
      renderPage();

      const input = screen.getByPlaceholderText(/filter integrations/i);
      input.focus();

      await user.keyboard("{ArrowDown}{ArrowDown}");
      await user.type(input, "git");

      expect(activeRowName()).toBe("github");
    });
  });
});
