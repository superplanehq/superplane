import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { DEFAULT_GITHUB_INTAKE_SETTINGS } from "./intakeSourceSettingsModel";

vi.mock("../layout/factoriesLayoutContext", () => ({
  useOptionalFactoriesLayout: () => ({ factoryId: "factory-1", routeSegment: "sp" }),
}));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ currentUserId: "alice" }) }));
vi.mock("@/hooks/useFactoryWebhooks", () => ({
  useFactoryWebhooks: () => ({
    data: [
      { id: "webhook-1", name: "Tracker bridge" },
      { id: "webhook-2", name: "Form bridge" },
    ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

function renderPopup(onSave: (next: unknown) => void) {
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              settings={{ ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Webhook", webhookId: "webhook-1" }}
              canEdit
              accessLoading={false}
              organizationId="org-1"
              sourceId="webhook"
              onSave={onSave}
              onClose={vi.fn()}
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("IntakeSourceSettingsPopup webhook fields", () => {
  it("saves the picked webhook and field paths", async () => {
    const onSave = vi.fn();
    renderPopup(onSave);
    const user = userEvent.setup();

    await user.click(screen.getByTestId("custom-automation-webhook-select"));
    await user.click(await screen.findByRole("option", { name: "Form bridge" }));
    await user.clear(screen.getByTestId("webhook-intake-settings-link-path"));
    await user.type(screen.getByTestId("webhook-intake-settings-link-path"), "issue.html_url");
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({
          webhookId: "webhook-2",
          webhookTitlePath: "title",
          webhookDescriptionPath: "description",
          webhookLinkPath: "issue.html_url",
        }),
      ),
    );
  });
});
