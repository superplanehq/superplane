import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { factoryLineDetailPath, factoryRouteSegment, firstFactoryLineId } from "../../lib/factoryPagePaths";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsGeneralPage } from "./FactorySettingsGeneralPage";

const mutateAsync = vi.fn();
const setVisibilityMutateAsync = vi.fn();
let canUpdate = true;

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1" } }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactory: () => ({ mutateAsync, isPending: false }),
  useDeleteFactory: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useSetFactoryVisibility: () => ({ mutateAsync: setVisibilityMutateAsync, isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdate, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: factory.id ?? "factory-1",
              factory,
            }}
          >
            <FactorySettingsGeneralPage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const badgeOnFactory = {
  ...REFUND_FACTORY,
  publicBadgeEnabled: true,
  publicBadgeShowCost: false,
  publicBadgeToken: "badge-token",
};

describe("FactorySettingsGeneralPage", () => {
  beforeEach(() => {
    canUpdate = true;
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({});
    setVisibilityMutateAsync.mockReset();
    setVisibilityMutateAsync.mockResolvedValue({});
  });

  it("shows a character avatar, name, and slug in a profile-style card", () => {
    renderPage();

    expect(screen.getByTestId("factory-settings-workspace-avatar")).toHaveTextContent("S");
    expect(screen.getByTestId("factory-settings-name")).toHaveValue("Semaphore");
    expect(screen.getByLabelText("Slug")).toHaveValue("RF");
    expect(screen.getByTestId("factory-settings-key")).toHaveValue("RF");
    expect(screen.queryByLabelText("Workspace key")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Description")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-settings-description")).not.toBeInTheDocument();
  });

  it("saves name and slug without sending a description", async () => {
    const user = userEvent.setup();
    renderPage();

    const save = screen.getByTestId("factory-settings-save");
    expect(save).toBeDisabled();

    await user.clear(screen.getByTestId("factory-settings-name"));
    await user.type(screen.getByTestId("factory-settings-name"), "Refunds");
    expect(screen.getByTestId("factory-settings-workspace-avatar")).toHaveTextContent("R");
    expect(save).toBeEnabled();

    await user.click(save);
    expect(mutateAsync).toHaveBeenCalledWith({ name: "Refunds" });
    expect(mutateAsync.mock.calls[0][0]).not.toHaveProperty("description");
  });

  it("shows the public badge toggle and saves it", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole("switch", { name: "Public badge" })).toBeInTheDocument();
    expect(screen.queryByRole("switch", { name: "Show cost per merged PR" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("switch", { name: "Public badge" }));
    expect(mutateAsync).toHaveBeenCalledWith({ publicBadgeEnabled: true });
  });

  it("shows the cost switch only when the badge is on and keeps the snippet stable", async () => {
    const user = userEvent.setup();
    mutateAsync.mockResolvedValue({ publicBadgeToken: "badge-token" });
    renderPage(badgeOnFactory);

    const cost = screen.getByRole("switch", { name: "Show cost per merged PR" });
    expect(cost).not.toBeChecked();

    const snippet = screen.getByTestId("factory-settings-public-badge-markdown");
    expect(snippet).toHaveValue(
      `[![PRs via SuperPlane](${window.location.origin}/api/v1/public/badges/badge-token.svg?period=30&size=small)](${window.location.origin})`,
    );

    await user.click(screen.getByTestId("factory-settings-public-badge-period"));
    await user.click(screen.getByRole("option", { name: "14 days" }));
    expect(snippet).toHaveValue(
      `[![PRs via SuperPlane](${window.location.origin}/api/v1/public/badges/badge-token.svg?period=14&size=small)](${window.location.origin})`,
    );

    await user.click(screen.getByTestId("factory-settings-public-badge-size"));
    await user.click(screen.getByRole("option", { name: "Full width" }));
    const afterSize = (snippet as HTMLInputElement).value;
    const wideURL = `${window.location.origin}/api/v1/public/badges/badge-token.svg?period=14&size=wide`;
    expect(afterSize).toBe(
      `<a href="${window.location.origin}"><img src="${wideURL}" alt="PRs via SuperPlane" width="100%"></a>`,
    );

    await user.click(cost);
    expect(mutateAsync).toHaveBeenCalledWith({ publicBadgeShowCost: true });
    expect(snippet).toHaveValue(afterSize);
  });

  it("disables badge controls without update permission", () => {
    canUpdate = false;
    renderPage(badgeOnFactory);

    expect(screen.getByRole("switch", { name: "Public badge" })).toBeDisabled();
    expect(screen.getByRole("switch", { name: "Show cost per merged PR" })).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-period")).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-size")).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-copy")).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-markdown")).toBeDisabled();
  });

  it("asks before it makes the workspace public", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("switch", { name: "Change to public" }));
    expect(screen.getByTestId("factory-settings-visibility-dialog")).toHaveTextContent("Make this workspace public?");
    expect(setVisibilityMutateAsync).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("factory-settings-visibility-cancel"));
    expect(setVisibilityMutateAsync).not.toHaveBeenCalled();

    await user.click(screen.getByRole("switch", { name: "Change to public" }));
    await user.click(screen.getByTestId("factory-settings-visibility-confirm"));
    expect(setVisibilityMutateAsync).toHaveBeenCalledWith(true);
  });

  it("shows the line board link when the workspace is public", () => {
    renderPage({ ...REFUND_FACTORY, public: true });

    const link = screen.getByTestId("factory-settings-visibility-board-link");
    expect(link).toHaveAttribute(
      "href",
      factoryLineDetailPath("org-1", factoryRouteSegment(REFUND_FACTORY), firstFactoryLineId(REFUND_FACTORY)!),
    );
    expect(link).toHaveTextContent("View public board");
  });

  it("loads a theme preset into the color fields and names it", async () => {
    const user = userEvent.setup();
    renderPage(badgeOnFactory);

    const snippet = screen.getByTestId("factory-settings-public-badge-markdown") as HTMLInputElement;
    expect(snippet.value).not.toContain("theme=");
    expect(screen.getByTestId("factory-settings-public-badge-theme-name")).toHaveTextContent("SuperPlane");

    await user.click(screen.getByRole("button", { name: "Tokyo Night" }));
    expect(snippet.value).toContain("theme=tokyonight");
    expect(screen.getByTestId("factory-settings-public-badge-theme-name")).toHaveTextContent("Tokyo Night");

    // The preset fills every field, and a preset alone adds no color params.
    await user.click(screen.getByTestId("factory-settings-public-badge-colors-toggle"));
    expect(screen.getByTestId("factory-settings-public-badge-color-bg")).toHaveValue("#1a1b27");
    expect(snippet.value).not.toContain("bg=");
  });

  it("shows the theme accent in the color swatch until the user changes it", async () => {
    const user = userEvent.setup();
    renderPage(badgeOnFactory);

    await user.click(screen.getByRole("button", { name: "Tokyo Night" }));
    await user.click(screen.getByTestId("factory-settings-public-badge-colors-toggle"));

    const swatch = screen.getByTestId("factory-settings-public-badge-color-swatch-accent");
    expect(swatch).toHaveValue("#7aa2f7");

    const snippet = screen.getByTestId("factory-settings-public-badge-markdown") as HTMLInputElement;
    expect(snippet.value).toContain("theme=tokyonight");
    expect(snippet.value).not.toContain("accent=");

    fireEvent.change(swatch, { target: { value: "#7aa2f7" } });
    expect(snippet.value).not.toContain("accent=");

    fireEvent.change(swatch, { target: { value: "#ff8800" } });
    expect(snippet.value).toContain("accent=ff8800");
    expect(swatch).toHaveValue("#ff8800");
  });

  it("sends only the colors that differ from the preset", async () => {
    const user = userEvent.setup();
    renderPage(badgeOnFactory);

    await user.click(screen.getByTestId("factory-settings-public-badge-colors-toggle"));
    const background = screen.getByTestId("factory-settings-public-badge-color-bg");
    await user.clear(background);
    await user.type(background, "#000000");

    const snippet = screen.getByTestId("factory-settings-public-badge-markdown") as HTMLInputElement;
    expect(snippet.value).toContain("bg=000000");
    expect(snippet.value).not.toContain("accent=");
    expect(screen.getByTestId("factory-settings-public-badge-theme-name")).toHaveTextContent("Custom");

    await user.click(screen.getByTestId("factory-settings-public-badge-reset"));
    expect((screen.getByTestId("factory-settings-public-badge-markdown") as HTMLInputElement).value).not.toContain(
      "bg=",
    );
  });

  it("ignores a color that is not a hex value", async () => {
    const user = userEvent.setup();
    renderPage(badgeOnFactory);

    await user.click(screen.getByTestId("factory-settings-public-badge-colors-toggle"));
    const accent = screen.getByTestId("factory-settings-public-badge-color-accent");
    await user.clear(accent);
    await user.type(accent, "nope");

    const snippet = screen.getByTestId("factory-settings-public-badge-markdown") as HTMLInputElement;
    expect(snippet.value).not.toContain("accent=");
  });

  it("explains on hover why the board link is off limits", async () => {
    const user = userEvent.setup();
    renderPage(badgeOnFactory);

    await user.hover(screen.getByTestId("factory-settings-public-badge-board-link"));
    const reason = await screen.findByText("The workspace is private. Make it public in the Visibility section above.");
    expect(reason).toBeInTheDocument();
  });

  it("links the badge to the board only while the workspace is public", async () => {
    const user = userEvent.setup();
    renderPage(badgeOnFactory);

    const privateSwitch = screen.getByRole("switch", { name: "Link the badge to the public board" });
    expect(privateSwitch).toBeDisabled();

    renderPage({ ...badgeOnFactory, public: true });
    const boardSwitch = screen
      .getAllByRole("switch", { name: "Link the badge to the public board" })
      .at(-1) as HTMLElement;
    expect(boardSwitch).not.toBeDisabled();

    await user.click(boardSwitch);
    const snippet = screen.getAllByTestId("factory-settings-public-badge-markdown").at(-1) as HTMLInputElement;
    expect(snippet.value).toContain(
      `](${window.location.origin}${factoryLineDetailPath("org-1", factoryRouteSegment(REFUND_FACTORY), firstFactoryLineId(REFUND_FACTORY)!)})`,
    );
  });
});
