import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { ComponentProps } from "react";

import dependabotIcon from "@/assets/icons/integrations/dependabot.svg";

import { BacklogCreatePopover } from "./BacklogCreatePopover";
import {
  BACKLOG_CREATE_COPY,
  searchPlaceholderForIntake,
  type BacklogIntakeItem,
  type BacklogIntakeSource,
} from "./backlogIntakeItems";

const sources: BacklogIntakeSource[] = [
  { intakeId: "intake-github", name: "GitHub issues", iconSrc: "/github.svg", iconAlt: "GitHub", tabLabel: "GitHub" },
  { intakeId: "intake-sentry", name: "Sentry exceptions", iconAlt: "Sentry", tabLabel: "Sentry" },
];

const githubItems: BacklogIntakeItem[] = [
  {
    id: "gh-1",
    intakeId: "intake-github",
    key: "#12",
    title: "Handle duplicate refunds",
    body: "Retrying a refund posts twice.",
  },
];

function popover(overrides: Partial<ComponentProps<typeof BacklogCreatePopover>> = {}) {
  return (
    <BacklogCreatePopover
      canAdd
      sources={sources}
      items={[]}
      query=""
      focusedIntakeId={null}
      onQueryChange={vi.fn()}
      onFocusedIntakeChange={vi.fn()}
      onCreateManually={vi.fn()}
      onImportItem={vi.fn()}
      {...overrides}
    />
  );
}

describe("BacklogCreatePopover", () => {
  it("shows source created times below titles and omits the line when missing", async () => {
    const recent = new Date(Date.now() - 5 * 60 * 1000);
    const old = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000);
    const items = [
      { ...githubItems[0], id: "recent", createdAt: recent.toISOString() },
      { ...githubItems[0], id: "old", createdAt: old.toISOString() },
      githubItems[0],
    ];
    const onImportItem = vi.fn();
    render(popover({ items, focusedIntakeId: "intake-github", onImportItem }));
    const user = userEvent.setup();
    await user.click(screen.getByTestId("lines-backlog-create"));

    const recentRow = screen.getByTestId("lines-backlog-create-item-recent");
    expect(within(recentRow).getByText("5 minutes ago")).toHaveAttribute("title", `Created ${recent.toLocaleString()}`);
    const oldRow = screen.getByTestId("lines-backlog-create-item-old");
    expect(oldRow.querySelector("time")).toHaveTextContent(String(old.getFullYear()));
    expect(oldRow.querySelector("time")).not.toHaveTextContent("ago");
    expect(oldRow.querySelector("time")).toHaveAttribute("title", `Created ${old.toLocaleString()}`);
    expect(screen.getByTestId("lines-backlog-create-item-gh-1").querySelector("time")).toBeNull();
    for (const row of [recentRow, oldRow]) {
      expect(within(row).getByText("#12")).toBeInTheDocument();
    }
    await user.click(recentRow);
    expect(onImportItem).toHaveBeenCalledWith(items[0]);
  });

  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn();
  });

  it("opens from the plus control and focuses the first intake", async () => {
    const onCreateManually = vi.fn();
    const onFocusedIntakeChange = vi.fn();
    const user = userEvent.setup();

    render(popover({ onCreateManually, onFocusedIntakeChange }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-menu").getAttribute("data-side")).toBe("right");
    expect(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually })).toBeInTheDocument();
    expect(screen.getByText(BACKLOG_CREATE_COPY.createManuallyHint)).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-tab-intake-github")).toHaveTextContent("GitHub");
    expect(screen.getByTestId("lines-backlog-create-tab-intake-sentry")).toHaveTextContent("Sentry");
    expect(screen.getByPlaceholderText(searchPlaceholderForIntake("GitHub issues"))).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(searchPlaceholderForIntake("Sentry exceptions"))).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-icon-intake-github")).toHaveAttribute("src", "/github.svg");
    expect(onFocusedIntakeChange).toHaveBeenCalledWith("intake-github");
    expect(screen.queryByTestId("lines-backlog-create-item-gh-1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-create-with-agent")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually }));
    expect(onCreateManually).toHaveBeenCalledTimes(1);
  });

  it("labels GitHub issues and Dependabot alerts as separate tabs", async () => {
    const user = userEvent.setup();
    const dependabot: BacklogIntakeSource = {
      intakeId: "intake-dependabot",
      name: "Dependabot alerts",
      iconSrc: dependabotIcon,
      iconAlt: "Dependabot",
      tabLabel: "Dependabot",
    };

    render(popover({ sources: [sources[0], dependabot], focusedIntakeId: "intake-dependabot" }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    const githubIcon = screen.getByTestId("lines-backlog-create-icon-intake-github");
    const dependabotMark = screen.getByTestId("lines-backlog-create-icon-intake-dependabot");
    expect(screen.getByTestId("lines-backlog-create-tab-intake-github")).toHaveTextContent("GitHub");
    expect(screen.getByRole("tab", { name: "Dependabot" })).toBeInTheDocument();
    expect(githubIcon).toHaveAttribute("src", "/github.svg");
    expect(dependabotMark).toHaveAttribute("src", dependabotIcon);
    expect(githubIcon).toHaveClass("dark:brightness-0", "dark:invert");
    expect(dependabotMark).toHaveClass("dark:brightness-0", "dark:invert");
    expect(screen.getByPlaceholderText("Import from Dependabot alert")).toBeInTheDocument();
  });

  it("does not open the create menu on hover", async () => {
    const user = userEvent.setup();

    render(popover());

    await user.hover(screen.getByTestId("lines-backlog-create"));
    expect(screen.queryByTestId("lines-backlog-create-menu")).not.toBeInTheDocument();
  });

  it("keeps the plus control in its hover style while the menu is open", async () => {
    const user = userEvent.setup();

    render(popover());

    const trigger = screen.getByTestId("lines-backlog-create");
    expect(trigger).not.toHaveClass("bg-accent");

    await user.click(trigger);
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    expect(trigger).toHaveClass("bg-accent");
    expect(trigger).toHaveClass("text-foreground");
  });

  it("keeps the ghost card in its hover style while the menu is open", async () => {
    const user = userEvent.setup();

    render(popover({ variant: "ghost" }));

    const trigger = screen.getByTestId("lines-backlog-create-ghost");
    expect(trigger).not.toHaveClass("bg-muted/70");

    await user.click(trigger);
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    expect(trigger).toHaveClass("bg-muted/70");
    expect(trigger).toHaveClass("text-foreground");
  });

  it("shows GitHub issues when the menu opens without a search click", async () => {
    const onFocusedIntakeChange = vi.fn();
    const onImportItem = vi.fn();
    const user = userEvent.setup();

    const { rerender } = render(popover({ onFocusedIntakeChange, onImportItem }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(onFocusedIntakeChange).toHaveBeenCalledWith("intake-github");

    rerender(
      popover({
        onFocusedIntakeChange,
        onImportItem,
        items: githubItems,
        focusedIntakeId: "intake-github",
      }),
    );

    expect(screen.getByTestId("lines-backlog-create-item-gh-1")).toHaveTextContent("Handle duplicate refunds");
    expect(screen.queryByTestId("lines-backlog-create-items-intake-sentry")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-backlog-create-item-gh-1"));
    expect(onImportItem).toHaveBeenCalledWith(githubItems[0]);
  });

  it("switches search and results to the selected intake tab", async () => {
    const onFocusedIntakeChange = vi.fn();
    const onImportItem = vi.fn();
    const sentryItems: BacklogIntakeItem[] = [
      {
        id: "se-1",
        intakeId: "intake-sentry",
        key: "PROJ-1",
        title: "Null pointer in checkout",
        body: "Checkout throws when the cart is empty.",
      },
    ];
    const user = userEvent.setup();

    const { rerender } = render(
      popover({ onFocusedIntakeChange, onImportItem, items: githubItems, focusedIntakeId: "intake-github" }),
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    await user.click(screen.getByTestId("lines-backlog-create-tab-intake-sentry"));
    expect(onFocusedIntakeChange).toHaveBeenCalledWith("intake-sentry");

    rerender(
      popover({
        onFocusedIntakeChange,
        onImportItem,
        items: sentryItems,
        focusedIntakeId: "intake-sentry",
      }),
    );

    expect(screen.getByPlaceholderText(searchPlaceholderForIntake("Sentry exceptions"))).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(searchPlaceholderForIntake("GitHub issues"))).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-item-se-1")).toHaveTextContent("Null pointer in checkout");
    expect(screen.queryByTestId("lines-backlog-create-items-intake-github")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-backlog-create-item-se-1"));
    expect(onImportItem).toHaveBeenCalledWith(sentryItems[0]);
  });

  it("resets results scroll when the intake tab changes", async () => {
    const onFocusedIntakeChange = vi.fn();
    const user = userEvent.setup();
    const manyGithubItems = Array.from({ length: 8 }, (_, index) => ({
      id: `gh-${index}`,
      intakeId: "intake-github",
      key: `#${index}`,
      title: `Issue ${index}`,
      body: "",
    }));
    const sentryItems: BacklogIntakeItem[] = [
      {
        id: "se-1",
        intakeId: "intake-sentry",
        key: "PROJ-1",
        title: "Null pointer in checkout",
        body: "",
      },
    ];

    const { rerender } = render(
      popover({ onFocusedIntakeChange, items: manyGithubItems, focusedIntakeId: "intake-github" }),
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    const githubList = screen.getByTestId("lines-backlog-create-items-intake-github");
    Object.defineProperty(githubList, "scrollHeight", { configurable: true, value: 400 });
    Object.defineProperty(githubList, "clientHeight", { configurable: true, value: 140 });
    githubList.scrollTop = 280;
    fireEvent.scroll(githubList);
    expect(githubList.scrollTop).toBe(280);

    await user.click(screen.getByTestId("lines-backlog-create-tab-intake-sentry"));
    expect(onFocusedIntakeChange).toHaveBeenCalledWith("intake-sentry");

    rerender(popover({ onFocusedIntakeChange, items: sentryItems, focusedIntakeId: "intake-sentry" }));

    const sentryList = screen.getByTestId("lines-backlog-create-items-intake-sentry");
    expect(sentryList.scrollTop).toBe(0);
  });

  it("scrolls extra source tabs inside the menu", async () => {
    const user = userEvent.setup();
    const longSources: BacklogIntakeSource[] = [
      { intakeId: "intake-linear", name: "Linear issues", iconAlt: "Linear", tabLabel: "Linear" },
      { intakeId: "intake-dependabot", name: "Dependabot alerts", iconAlt: "Dependabot", tabLabel: "Dependabot" },
      { intakeId: "intake-datadog", name: "Datadog errors", iconAlt: "Datadog", tabLabel: "Datadog" },
      { intakeId: "intake-sentry", name: "Sentry exceptions", iconAlt: "Sentry", tabLabel: "Sentry" },
    ];

    render(popover({ sources: longSources, focusedIntakeId: "intake-linear" }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    const tabs = screen.getByTestId("lines-backlog-create-tabs");
    expect(tabs.className).toContain("max-w-full");
    expect(tabs.className).toContain("overflow-x-auto");
    expect(screen.getByTestId("lines-backlog-create-menu")).toHaveClass("w-96");
    for (const source of longSources) {
      expect(screen.getByTestId(`lines-backlog-create-tab-${source.intakeId}`)).toHaveTextContent(source.tabLabel);
    }
  });

  it("keeps a single intake as a search row without tabs", async () => {
    const user = userEvent.setup();

    render(popover({ sources: [sources[0]], focusedIntakeId: "intake-github" }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.queryByTestId("lines-backlog-create-tabs")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-create-tab-intake-github")).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText(searchPlaceholderForIntake("GitHub issues"))).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-icon-intake-github")).toBeInTheDocument();
  });

  it("loads the next search page when the results list is scrolled to the end", async () => {
    const onLoadMore = vi.fn();
    const user = userEvent.setup();
    const manyItems = Array.from({ length: 5 }, (_, index) => ({
      id: `gh-${index}`,
      intakeId: "intake-github",
      key: `#${index}`,
      title: `Issue ${index}`,
      body: "",
    }));

    render(popover({ items: manyItems, focusedIntakeId: "intake-github", hasMore: true, onLoadMore }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    const list = screen.getByTestId("lines-backlog-create-items-intake-github");
    Object.defineProperty(list, "scrollHeight", { configurable: true, value: 400 });
    Object.defineProperty(list, "clientHeight", { configurable: true, value: 140 });
    list.scrollTop = 280;
    fireEvent.scroll(list);

    expect(onLoadMore).toHaveBeenCalled();
  });

  it("shows a spinner while the first search page loads", async () => {
    const user = userEvent.setup();
    render(popover({ focusedIntakeId: "intake-github", isLoading: true }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    const status = screen.getByTestId("lines-backlog-create-loading");
    expect(status).toHaveTextContent(BACKLOG_CREATE_COPY.loading);
    expect(status.querySelector("svg.animate-spin")).not.toBeNull();
  });

  it("shows a spinner while the next search page loads", async () => {
    const user = userEvent.setup();
    render(popover({ items: githubItems, focusedIntakeId: "intake-github", isLoadingMore: true, hasMore: true }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    const status = screen.getByTestId("lines-backlog-create-loading-more");
    expect(status).toHaveTextContent(BACKLOG_CREATE_COPY.loadingMore);
    expect(status.querySelector("svg.animate-spin")).not.toBeNull();
  });

  it("keeps search results when a later page fails", async () => {
    const user = userEvent.setup();
    render(
      popover({
        items: githubItems,
        focusedIntakeId: "intake-github",
        errorMessage: BACKLOG_CREATE_COPY.unconnected,
      }),
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-item-gh-1")).toHaveTextContent("Handle duplicate refunds");
    expect(screen.queryByText(BACKLOG_CREATE_COPY.unconnected)).not.toBeInTheDocument();
  });

  it("shows the search error when the intake is not connected", async () => {
    const user = userEvent.setup();
    render(popover({ focusedIntakeId: "intake-github", errorMessage: BACKLOG_CREATE_COPY.unconnected }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-items-intake-github")).toHaveTextContent(
      BACKLOG_CREATE_COPY.unconnected,
    );
  });

  it("opens the create menu when no intakes are configured", async () => {
    const onCreateManually = vi.fn();
    const user = userEvent.setup();
    render(popover({ sources: [], onCreateManually }));

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually }));
    expect(onCreateManually).toHaveBeenCalledTimes(1);
  });

  it("opens the create menu from the ghost card", async () => {
    const user = userEvent.setup();
    render(popover({ variant: "ghost" }));

    await user.click(screen.getByTestId("lines-backlog-create-ghost"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually })).toBeInTheDocument();
  });

  it("does not open the ghost card when the backlog cannot accept work", async () => {
    const user = userEvent.setup();
    render(popover({ canAdd: false, variant: "ghost" }));

    expect(screen.getByTestId("lines-backlog-create-ghost")).toBeDisabled();
    await user.click(screen.getByTestId("lines-backlog-create-ghost"));
    expect(screen.queryByTestId("lines-backlog-create-menu")).not.toBeInTheDocument();
  });

  it("does not open when the backlog cannot accept work", async () => {
    const user = userEvent.setup();
    render(popover({ canAdd: false }));

    expect(screen.getByTestId("lines-backlog-create")).toBeDisabled();
    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.queryByTestId("lines-backlog-create-menu")).not.toBeInTheDocument();
  });
});
