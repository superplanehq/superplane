import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { BacklogCreatePopover } from "./BacklogCreatePopover";
import {
  BACKLOG_CREATE_COPY,
  searchPlaceholderForIntake,
  type BacklogIntakeItem,
  type BacklogIntakeSource,
} from "./backlogIntakeItems";

const sources: BacklogIntakeSource[] = [
  { intakeId: "intake-github", name: "GitHub issues", iconSrc: "/github.svg", iconAlt: "GitHub" },
  { intakeId: "intake-sentry", name: "Sentry exceptions", iconAlt: "Sentry" },
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

describe("BacklogCreatePopover", () => {
  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn();
  });

  it("opens from the plus control with create and collapsed intake searches", async () => {
    const onCreateManually = vi.fn();
    const onImportItem = vi.fn();
    const onQueryChange = vi.fn();
    const onFocusedIntakeChange = vi.fn();
    const user = userEvent.setup();

    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={onQueryChange}
        onFocusedIntakeChange={onFocusedIntakeChange}
        onCreateManually={onCreateManually}
        onImportItem={onImportItem}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-menu")).toHaveAttribute("data-side", "right");
    expect(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually })).toBeInTheDocument();
    expect(screen.getByPlaceholderText(searchPlaceholderForIntake("GitHub issues"))).toBeInTheDocument();
    expect(screen.getByPlaceholderText(searchPlaceholderForIntake("Sentry exceptions"))).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-create-icon-intake-github")).toHaveAttribute("src", "/github.svg");
    expect(screen.queryByTestId("lines-backlog-create-item-gh-1")).not.toBeInTheDocument();

    expect(screen.queryByTestId("lines-backlog-create-with-agent")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually }));
    expect(onCreateManually).toHaveBeenCalledTimes(1);
  });

  it("starts an agent session from the create menu", async () => {
    const onCreateWithAgent = vi.fn();
    const user = userEvent.setup();

    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onCreateWithAgent={onCreateWithAgent}
        onImportItem={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    await user.click(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createWithAgent }));
    expect(onCreateWithAgent).toHaveBeenCalledTimes(1);
  });

  it("expands a few issues when an intake search is selected", async () => {
    const onFocusedIntakeChange = vi.fn();
    const onImportItem = vi.fn();
    const user = userEvent.setup();

    const { rerender } = render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={onFocusedIntakeChange}
        onCreateManually={vi.fn()}
        onImportItem={onImportItem}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    await user.click(screen.getByPlaceholderText(searchPlaceholderForIntake("GitHub issues")));
    expect(onFocusedIntakeChange).toHaveBeenCalledWith("intake-github");

    rerender(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={githubItems}
        query=""
        focusedIntakeId="intake-github"
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={onFocusedIntakeChange}
        onCreateManually={vi.fn()}
        onImportItem={onImportItem}
      />,
    );

    expect(screen.getByTestId("lines-backlog-create-item-gh-1")).toHaveTextContent("Handle duplicate refunds");
    expect(screen.queryByTestId("lines-backlog-create-items-intake-sentry")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-backlog-create-item-gh-1"));
    expect(onImportItem).toHaveBeenCalledWith(githubItems[0]);
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

    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={manyItems}
        query=""
        focusedIntakeId="intake-github"
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
        hasMore
        onLoadMore={onLoadMore}
      />,
    );

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
    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId="intake-github"
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
        isLoading
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    const status = screen.getByTestId("lines-backlog-create-loading");
    expect(status).toHaveTextContent(BACKLOG_CREATE_COPY.loading);
    expect(status.querySelector("svg.animate-spin")).not.toBeNull();
  });

  it("shows a spinner while the next search page loads", async () => {
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={githubItems}
        query=""
        focusedIntakeId="intake-github"
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
        isLoadingMore
        hasMore
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    const status = screen.getByTestId("lines-backlog-create-loading-more");
    expect(status).toHaveTextContent(BACKLOG_CREATE_COPY.loadingMore);
    expect(status.querySelector("svg.animate-spin")).not.toBeNull();
  });

  it("keeps search results when a later page fails", async () => {
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={githubItems}
        query=""
        focusedIntakeId="intake-github"
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
        errorMessage={BACKLOG_CREATE_COPY.unconnected}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-item-gh-1")).toHaveTextContent("Handle duplicate refunds");
    expect(screen.queryByText(BACKLOG_CREATE_COPY.unconnected)).not.toBeInTheDocument();
  });

  it("shows the search error when the intake is not connected", async () => {
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId="intake-github"
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
        errorMessage={BACKLOG_CREATE_COPY.unconnected}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-items-intake-github")).toHaveTextContent(
      BACKLOG_CREATE_COPY.unconnected,
    );
  });

  it("opens the create menu when no intakes are configured", async () => {
    const onCreateManually = vi.fn();
    const onCreateWithAgent = vi.fn();
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd
        sources={[]}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={onCreateManually}
        onCreateWithAgent={onCreateWithAgent}
        onImportItem={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually }));
    expect(onCreateManually).toHaveBeenCalledTimes(1);
  });

  it("opens the create menu from the ghost card", async () => {
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd
        variant="ghost"
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("lines-backlog-create-ghost"));
    expect(screen.getByTestId("lines-backlog-create-menu")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: BACKLOG_CREATE_COPY.createManually })).toBeInTheDocument();
  });

  it("does not open the ghost card when the backlog cannot accept work", async () => {
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd={false}
        variant="ghost"
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
      />,
    );

    expect(screen.getByTestId("lines-backlog-create-ghost")).toBeDisabled();
    await user.click(screen.getByTestId("lines-backlog-create-ghost"));
    expect(screen.queryByTestId("lines-backlog-create-menu")).not.toBeInTheDocument();
  });

  it("does not open when the backlog cannot accept work", async () => {
    const user = userEvent.setup();
    render(
      <BacklogCreatePopover
        canAdd={false}
        sources={sources}
        items={[]}
        query=""
        focusedIntakeId={null}
        onQueryChange={vi.fn()}
        onFocusedIntakeChange={vi.fn()}
        onCreateManually={vi.fn()}
        onImportItem={vi.fn()}
      />,
    );

    expect(screen.getByTestId("lines-backlog-create")).toBeDisabled();
    await user.click(screen.getByTestId("lines-backlog-create"));
    expect(screen.queryByTestId("lines-backlog-create-menu")).not.toBeInTheDocument();
  });
});
