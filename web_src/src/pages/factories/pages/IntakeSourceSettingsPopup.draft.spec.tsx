import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSourceSettings } from "./intakeSourceSettingsModel";

vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ currentUserId: "alice" }) }));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

function popup(
  settings: IntakeSourceSettings,
  labelOptions: string[],
  onSave: (next: IntakeSourceSettings) => void,
  organizationId?: string,
  sourceId: "github-issues" | "dependabot-alerts" = "github-issues",
) {
  return (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              settings={settings}
              organizationId={organizationId}
              sourceId={sourceId}
              labelOptions={labelOptions}
              onSave={onSave}
              onClose={vi.fn()}
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

async function pickLabel(name: string) {
  const user = userEvent.setup();
  await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));
  await user.click(within(screen.getByTestId("intake-label-options")).getByRole("checkbox", { name }));
  return user;
}

describe("IntakeSourceSettingsPopup draft", () => {
  it("does not offer a default owner for other intake sources", () => {
    render(popup(DEFAULT_GITHUB_INTAKE_SETTINGS, [], vi.fn(), "org-owner", "dependabot-alerts"));
    expect(screen.queryByRole("button", { name: "Default task owner" })).not.toBeInTheDocument();
  });
  it("keeps the selected owner in the draft until Save", async () => {
    queryClient.setQueryData(["organization", "users", "org-owner"], {
      users: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    });
    const onSave = vi.fn();
    render(popup(DEFAULT_GITHUB_INTAKE_SETTINGS, [], onSave, "org-owner"));
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Default task owner" }));
    await user.click(await screen.findByRole("option", { name: "Alice" }));
    expect(onSave).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("intake-source-settings-save"));
    await waitFor(() => expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ defaultOwnerId: "alice" })));
  });
  it("saves a label picked immediately before Save", async () => {
    const onSave = vi.fn();
    render(popup(DEFAULT_GITHUB_INTAKE_SETTINGS, ["bug"], onSave));

    const user = await pickLabel("bug");
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ labels: ["bug"], filterByLabel: true })),
    );
  });

  it("keeps unsaved label edits when the settings prop gets a new object reference", async () => {
    const onSave = vi.fn();
    const initial = { ...DEFAULT_GITHUB_INTAKE_SETTINGS };
    const { rerender } = render(popup(initial, ["bug", "enhancement"], onSave));

    const user = await pickLabel("bug");
    rerender(popup({ ...initial }, ["bug", "enhancement", "needs design"], onSave));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ labels: ["bug"], filterByLabel: true })),
    );
  });
});
