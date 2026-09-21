import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as FactoryData from "@/hooks/useFactoryData";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

const factoryPlanning = { current: { enabled: true, clarity: true, confidence: true } };
const mergeability = {
  current: {
    canMerge: true,
    allowedMethods: ["MERGE_METHOD_SQUASH", "MERGE_METHOD_REBASE"],
    headSha: "abc123",
  } as {
    canMerge: boolean;
    blockedReason?: string;
    message?: string;
    allowedMethods: string[];
    headSha: string;
  },
};
const mergeMutate = vi.fn();

vi.mock("@/hooks/useFactoryData", () => {
  const actual = unmockedSrc<typeof FactoryData>("hooks/useFactoryData");
  return {
    ...actual,
    useFactory: () => ({
      data: { id: "factory-1", planning: factoryPlanning.current },
      isPending: false,
    }),
  };
});

vi.mock("@/hooks/useFactoryPullRequestMerge", () => ({
  useFactoryPullRequestMergeability: () => ({ data: mergeability.current }),
  useMergeFactoryPullRequest: () => ({ mutate: mergeMutate, isPending: false }),
}));

import { factoryAppSplitRunPath } from "../../lib/factoryPagePaths";
import {
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import {
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  BOARD_IMPLEMENT_NOTIFY_ORDER,
} from "../../__fixtures__/lineMetricsBoardOrders";
import {
  LINE_BOARD_DONE_RECEIPTS_ORDER,
  LINE_BOARD_VERIFY_ENUM_ORDER,
} from "../../__fixtures__/lineMetricsFactoriesFixture";
import { OPEN_WORK_ORDER_CHECKS, VERIFY_STEP_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { SPEC_ARTIFACT_NAME } from "../../lib/intentDocument";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { buildSplitRunFooter } from "./splitRunFooter";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder, type SplitRunFixture } from "./splitRunMocks";
import { SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";
import { WORK_ORDER_FULL_PAGE_STORAGE_KEY } from "./workOrderFullPagePreference";

function withPublishedSpec(fixture: SplitRunFixture): SplitRunFixture {
  const spec = {
    id: "art-spec-test",
    type: "TYPE_MARKDOWN" as const,
    data: {
      name: SPEC_ARTIFACT_NAME,
      title: SPEC_ARTIFACT_NAME,
      body: "# Spec\n\n## Executive summary\n\nA published plan.\n",
    },
  };
  const [first, ...rest] = fixture.phases;
  if (!first) {
    return fixture;
  }
  return { ...fixture, phases: [{ ...first, artifacts: [...first.artifacts, spec] }, ...rest] };
}

function fixtureWithReviewPullRequest(state: "STATE_OPEN" | "STATE_MERGED"): SplitRunFixture {
  const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER);
  const [first, ...rest] = fixture.phases;
  if (!first) {
    return fixture;
  }
  return {
    ...fixture,
    phases: [
      {
        ...first,
        stream: [
          ...first.stream,
          {
            id: "pr-review-line",
            at: "now",
            componentName: "Pull request",
            status: "passed",
            pullRequest: {
              id: "pr-open",
              provider: "PROVIDER_GITHUB",
              url: "https://github.com/superplanehq/superplane/pull/6812",
              number: "6812",
              state,
            },
          },
        ],
      },
      ...rest,
    ],
  };
}

function renderPopup(props: ComponentProps<typeof WorkOrderSplitRunPopup>) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup {...props} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderSplitRun() {
  return renderPopup({ fixture: SPLIT_RUN_RUNNING });
}

async function openLogTab(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("tab", { name: "Automations" }));
}

describe("WorkOrderSplitRunPopup", () => {
  beforeEach(() => {
    window.localStorage.clear();
    factoryPlanning.current = { enabled: true, clarity: true, confidence: true };
    mergeMutate.mockReset();
    mergeability.current = {
      canMerge: true,
      allowedMethods: ["MERGE_METHOD_SQUASH", "MERGE_METHOD_REBASE"],
      headSha: "abc123",
    };
  });

  it("does not put an expand control on the Log heading", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    expect(screen.queryByTestId("split-run-log-expand")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open automation run" })).not.toBeInTheDocument();
  });

  it("separates task automations from pull request activity", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-fb",
            handlerName: "Address PR feedback",
            pullRequestNumber: "12",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            run: {
              id: "run-fb",
              canvasId: "canvas-fb",
              state: "STATE_STARTED",
              result: "RESULT_UNKNOWN",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("split-run-log-tab-dot")).toHaveAttribute("data-status-mark", "running");
    expect(screen.getByTestId("split-run-log-tab-dot")).toHaveClass("animate-spin");
    await openLogTab(user);
    expect(screen.queryByRole("heading", { name: "Task automations" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Pull request activity" })).not.toBeInTheDocument();
    const heading = screen.getByRole("heading", {
      name: "#12 feat: add endpoint to re-shuffle an existing deck",
    });
    const titleLink = within(heading).getByRole("link", {
      name: "#12 feat: add endpoint to re-shuffle an existing deck",
    });
    expect(titleLink).toHaveAttribute("href", "https://github.com/example/repo/pull/12");
    expect(titleLink).not.toHaveClass("w-full");
    expect(heading.querySelector(".lucide-activity")).toBeInTheDocument();
    expect(heading.querySelector(".lucide-git-pull-request")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-pr-feedback-run-fb")).toHaveTextContent("Activity on PR #12");
    expect(
      within(screen.getByTestId("split-run-task-automations")).getAllByTestId(/^split-run-phase-time-/).length,
    ).toBeGreaterThan(0);
  });

  it("keeps a queued activity title and uses the waiting clock", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-fb",
            title:
              "[@lucaspin](https://github.com/lucaspin) left a [review](https://github.com/acme/app/pull/12#pullrequestreview-1)",
            description: "Read the requested changes.",
            waitingForAccess: true,
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            run: {
              id: "run-queued",
              canvasId: "canvas-fb",
              state: "STATE_STARTED",
              result: "RESULT_UNKNOWN",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("split-run-log-tab-dot")).toHaveAttribute("data-status-mark", "running");
    const activity = screen.getByTestId("split-run-phase-pr-feedback-run-queued");
    expect(within(activity).getByRole("link", { name: "@lucaspin" })).toBeInTheDocument();
    expect(within(activity).getByRole("link", { name: "review" })).toBeInTheDocument();
    expect(activity).toHaveTextContent("Read the requested changes.");
    expect(activity.querySelector(".lucide-clock")).toBeInTheDocument();
    expect(activity.querySelector(".lucide-loader-circle")).toBeNull();
    await user.hover(within(activity).getByLabelText("Waiting for another activity"));
    expect(await screen.findByRole("tooltip", { name: "Waiting for another activity" })).toBeInTheDocument();
  });

  it("shows pull request activity as a flat timestamp-first timeline", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-fb",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            title:
              "[@lucaspin](https://github.com/lucaspin) left a [comment](https://github.com/acme/app/pull/12#issuecomment-1)",
            description: "Read the [requested changes](https://example.com/review).",
            run: {
              id: "run-comment",
              canvasId: "canvas-fb",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
          {
            canvasId: "canvas-checks",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            revision: { id: "revision-a", sha: "a82fd91" },
            title: "Wait for checks",
            run: {
              id: "run-checks",
              canvasId: "canvas-checks",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T10:00:00Z",
            },
          },
          {
            canvasId: "canvas-fb",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            revision: { id: "revision-b", sha: "d8b80c2" },
            title: "Address new review comment",
            run: {
              id: "run-new-revision",
              canvasId: "canvas-fb",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T12:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.getByRole("tab", { name: "Task" })).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("split-run-log-tab-dot")).not.toHaveAttribute("data-status-mark", "running");
    expect(screen.getByTestId("split-run-log-tab-dot")).not.toHaveClass("animate-spin");
    await openLogTab(user);

    const pullRequestActivity = screen.getByTestId("split-run-pull-request-activity");
    expect(within(pullRequestActivity).getAllByRole("link", { name: /#12/ })).toHaveLength(1);

    const timeline = screen.getByTestId("split-run-pull-request-timeline-0");
    expect(timeline).not.toHaveClass("border-l");
    const activities = within(timeline).getAllByTestId(/^split-run-pull-request-activity-item-/);
    expect(activities).toHaveLength(3);
    expect(
      activities.map((activity) => within(activity).getByTestId(/^split-run-phase-pr-feedback-/).textContent),
    ).toEqual([
      expect.stringContaining("Wait for checks"),
      expect.stringContaining("@lucaspin left a comment"),
      expect.stringContaining("Address new review comment"),
    ]);
    expect(
      within(timeline)
        .getAllByTestId(/^split-run-phase-time-pr-feedback-/)
        .map((time) => time.getAttribute("datetime")),
    ).toEqual(["2026-08-26T10:00:00Z", "2026-08-26T11:00:00Z", "2026-08-26T12:00:00Z"]);
    expect(within(timeline).queryByTestId(/^split-run-phase-revision-/)).not.toBeInTheDocument();
    const commentTime = within(activities[1]!).getByTestId("split-run-phase-time-pr-feedback-run-comment");
    expect(commentTime.closest("[data-testid='split-run-automation-header-pr-feedback-run-comment']")).toBe(
      within(activities[1]!).getByTestId("split-run-automation-header-pr-feedback-run-comment"),
    );
    const authorLink = within(activities[1]!).getByRole("link", { name: "@lucaspin" });
    const commentLink = within(activities[1]!).getByRole("link", { name: "comment" });
    const descriptionLink = within(activities[1]!).getByRole("link", { name: "requested changes" });
    expect(authorLink.closest(".workspace-markdown")).toHaveClass("font-mono", "text-[13px]");
    expect(within(activities[1]!).queryByRole("button", { name: /^(Expand|Collapse) / })).not.toBeInTheDocument();
    expect(
      within(activities[1]!).getByTestId("split-run-automation-header-pr-feedback-run-comment").className,
    ).not.toMatch(/\bh-8\b/);
    expect(within(activities[1]!).getByTestId("split-run-phase-description-pr-feedback-run-comment")).toHaveClass(
      "font-mono",
      "text-[13px]",
    );
    expect(within(activities[1]!).getByTestId("split-run-phase-description-pr-feedback-run-comment")).toHaveTextContent(
      "Read the requested changes",
    );
    expect(within(activities[1]!).queryByTestId("split-run-stream-pr-feedback-run-comment")).not.toBeInTheDocument();
    expect(authorLink).toHaveAttribute("href", "https://github.com/lucaspin");
    expect(commentLink).toHaveAttribute("href", "https://github.com/acme/app/pull/12#issuecomment-1");
    expect(descriptionLink).toHaveAttribute("href", "https://example.com/review");
    for (const link of [authorLink, commentLink, descriptionLink]) {
      expect(link).toHaveAttribute("target", "_blank");
      expect(link).toHaveAttribute("rel", "noopener noreferrer");
      expect(link).toHaveClass("font-semibold", "text-current", "!underline", "!decoration-current");
    }
  });

  it("keeps the log scroller flush so sticky phase headers cover scrolled lines", () => {
    renderSplitRun();
    const scroll = screen.getByTestId("split-run-log-scroll");
    expect(scroll.className).not.toMatch(/\bpy-\d/);
    expect(scroll.className).not.toMatch(/\bpt-\d/);
    expect(scroll.className).toMatch(/\bpb-3\b/);
  });

  it("does not show elapsed time or a spend icon on the owner row", () => {
    renderSplitRun();
    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
    const row = screen.getByTestId("popup-owner-time-cost");
    expect(within(row).queryByText(/so far/)).not.toBeInTheDocument();
    expect(row.querySelector(".lucide-clock")).toBeNull();
    expect(row.querySelector(".lucide-circle-dollar-sign")).toBeNull();
    expect(row).toHaveTextContent("$0.73");
    expect(row).toHaveTextContent("2.7k tokens");
    expect(row).not.toHaveTextContent("claude-sonnet-4-6");
    expect(within(row).queryByRole("tablist")).not.toBeInTheDocument();
    const close = screen.getByRole("button", { name: "Close" });
    const views = screen.getByRole("tablist", { name: "Task views" });
    expect(close.compareDocumentPosition(views) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(views).getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    expect(within(views).getByRole("tab", { name: "Automations" })).toHaveClass("sp-popup-view-tab");
  });

  it("keeps the implement model off the header line while the task is running", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER) });

    const row = screen.getByTestId("popup-owner-time-cost");
    expect(row).toHaveTextContent("$0.73 · 2.7k tokens");
    expect(row).not.toHaveTextContent("claude-sonnet-4-6");
    expect(within(row).queryByRole("combobox")).not.toBeInTheDocument();
  });

  it("underlines header spend and shows a model breakdown on hover", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER) });

    const trigger = screen.getByTestId("popup-spend-breakdown-trigger");
    expect(trigger).toHaveClass("underline");
    await user.hover(trigger);
    const card = await screen.findByTestId("popup-spend-breakdown");
    expect(card).toHaveTextContent("claude-sonnet-4-6");
    expect(card).toHaveTextContent("2.7k tokens · $0.45");
    expect(card).toHaveTextContent("Machine time");
    expect(card).toHaveTextContent("1 min 30 s · $0.28");
  });

  it("does not show a model on a draft that has not started", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    const row = screen.getByTestId("popup-owner-time-cost");
    expect(row).toHaveTextContent("$0.00");
    expect(row).toHaveTextContent("0 tokens");
    expect(row).not.toHaveTextContent("claude-sonnet-4-6");
    expect(row).not.toHaveTextContent("grok-4.6");
    expect(screen.queryByTestId("popup-spend-breakdown-trigger")).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-spend-breakdown")).not.toBeInTheDocument();
  });

  it("opens the Task tab when no automation is running", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER) });

    const row = screen.getByTestId("popup-owner-time-cost");
    expect(within(row).queryByRole("tablist")).not.toBeInTheDocument();
    const views = screen.getByRole("tablist", { name: "Task views" });
    expect(
      screen.getByRole("button", { name: "Close" }).compareDocumentPosition(views) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(within(views).getByRole("tab", { name: "Task" })).toHaveAttribute("data-state", "active");
    expect(within(views).getByRole("tab", { name: "Task" })).toHaveClass("sp-popup-view-tab");
    expect(within(views).getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "inactive");
  });

  it("shows tokens and cost on a line-step phase", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER, { demoArtifacts: false }) });

    expect(screen.getByTestId("split-run-phase-duration-implement-0")).toHaveTextContent(
      "$0.28 · 900 · claude-sonnet-4-6 ·",
    );
  });

  it("does not put an Open task link next to close", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
    });

    expect(screen.queryByTestId("split-run-open-work-order")).not.toBeInTheDocument();

    expect(screen.queryByRole("link", { name: "Open task" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("expands next to Close into a full page that leaves the sidebar uncovered", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      fixed: true,
    });

    const expand = screen.getByRole("button", { name: "Open full screen" });
    const close = screen.getByRole("button", { name: "Close" });
    expect(expand.compareDocumentPosition(close) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(expand).toHaveClass("h-6", "w-6", "rounded-full");
    expect(close).toHaveClass("h-6", "w-6", "rounded-full");

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className.split(/\s+/)).toEqual(
      expect.arrayContaining(SPLIT_RUN_POPUP_DIALOG_CLASSNAME.split(/\s+/)),
    );
    expect(dialog.parentElement).toHaveClass("fixed");
    expect(dialog.parentElement?.className).not.toContain("left-[var(--workspace-navigation-width)]");

    await user.click(expand);

    const fullPage = screen.getByTestId("work-order-split-run");
    expect(fullPage.className).toContain("h-full");
    expect(fullPage.className).toContain("w-full");
    expect(fullPage.className).not.toContain("w-[min(70rem");
    expect(fullPage.parentElement).toHaveClass("fixed");
    expect(fullPage.parentElement?.className).toContain("left-[var(--workspace-navigation-width)]");
    expect(fullPage.parentElement).not.toHaveClass("bg-black/50");
    expect(screen.getByRole("button", { name: "Exit full screen" })).toBeInTheDocument();

    expect(window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY)).toBe("1");

    await user.click(screen.getByRole("button", { name: "Exit full screen" }));

    expect(screen.getByTestId("work-order-split-run").className.split(/\s+/)).toEqual(
      expect.arrayContaining(SPLIT_RUN_POPUP_DIALOG_CLASSNAME.split(/\s+/)),
    );
    expect(screen.getByRole("button", { name: "Open full screen" })).toBeInTheDocument();
    expect(window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY)).toBe("0");
  });

  it("opens the task modal in full screen from the stored preference", () => {
    window.localStorage.setItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY, "1");
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      fixed: true,
    });

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className).toContain("h-full");
    expect(dialog.className).toContain("w-full");
    expect(screen.getByRole("button", { name: "Exit full screen" })).toBeInTheDocument();
  });

  it("collapses finished steps and expands the running component stream", () => {
    renderSplitRun();

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className.split(/\s+/)).toEqual(
      expect.arrayContaining(SPLIT_RUN_POPUP_DIALOG_CLASSNAME.split(/\s+/)),
    );
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Task" })).toBeInTheDocument();
    const runningDot = within(dialog).getByTestId("split-run-log-tab-dot");
    expect(runningDot).toHaveAttribute("title", "Running");
    expect(runningDot.className).toContain("animate-spin");
    expect(within(dialog).queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Automations" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-log-scroll")).toBeInTheDocument();
    expect(within(dialog).queryByRole("region", { name: "Run" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();

    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).getByText("Backlog")).toBeInTheDocument();
    expect(within(backlog).getByTestId("split-run-phase-duration-backlog")).toHaveTextContent("2s");
    expect(within(backlog).getByRole("button", { name: "description.md" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();

    const implement = screen.getByTestId("split-run-phase-implement");
    expect(within(implement).getAllByText(/Implementation/).length).toBeGreaterThan(0);
    expect(within(implement).getByTestId("split-run-phase-duration-implement")).toHaveTextContent("4m");
    expect(within(implement).getAllByRole("link", { name: /feature\/refund-retry/ }).length).toBeGreaterThan(0);
    expect(screen.getByTestId("split-run-stream-implement")).toBeInTheDocument();
    expect(within(implement).queryByText("Started")).not.toBeInTheDocument();
    expect(within(implement).getAllByText("Start Implementation").length).toBeGreaterThan(0);
    expect(
      within(screen.getByTestId("split-run-stream-line-onrun-implement")).queryByText("On Run"),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-line-onrun-implement")).not.toHaveTextContent(/\d{2}:\d{2}:\d{2}/);
    expect(
      within(screen.getByTestId("split-run-stream-line-onrun-implement")).queryByText(">"),
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-line-onrun-implement")).queryByText("├──"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-line-onrun-implement")).queryByText("└──"),
    ).not.toBeInTheDocument();
    const implementStream = screen.getByTestId("split-run-stream-implement");
    const note = within(implementStream).getByText("Provide Plan");
    expect(note.closest("li")).not.toHaveTextContent("├──");
    expect(note.closest("li")).not.toHaveTextContent("└──");
    expect(note.closest("li")).toHaveTextContent("bash");
    expect(within(implementStream).getByText("Set Up Environment")).toBeInTheDocument();
    expect(within(implementStream).getAllByText("✓").length).toBeGreaterThan(0);
    expect(within(implementStream).getByText(/superplaneagent@superplane.com/)).toBeInTheDocument();
    expect(
      within(implementStream).getByText(
        "Now let's look at the messages file, factory_notification_consumer.go, and other referenced files.",
      ),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-implement")).queryByText("├──")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-implement")).queryByText("└──")).not.toBeInTheDocument();
    expect(within(implement).queryByText("did not run")).not.toBeInTheDocument();

    expect(screen.getAllByText("Implementation").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
  });

  it("shows produced artifacts on the automation line", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    const implement = screen.getByTestId("split-run-phase-implement");
    const implementArtifacts = within(implement).getByTestId("split-run-phase-artifacts-implement");
    expect(implementArtifacts.parentElement?.className).toMatch(/ml-auto/);
    expect(within(implementArtifacts).getByRole("link", { name: /feature\/refund-retry/ })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-implement")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-stream-implement")).getByRole("link", { name: /feature\/refund-retry/ }),
    ).toBeInTheDocument();

    await user.click(within(implement).getByTestId("split-run-automation-header-implement"));

    expect(screen.queryByTestId("split-run-stream-implement")).not.toBeInTheDocument();
    expect(within(implement).getByTestId("split-run-phase-artifacts-implement")).toBeInTheDocument();
    expect(within(implement).getByRole("link", { name: /feature\/refund-retry/ })).toBeInTheDocument();
  });

  it("shows canvas artifacts on a collapsed automation before it is opened", () => {
    renderSplitRun();

    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(screen.queryByTestId("split-run-stream-backlog")).not.toBeInTheDocument();
    expect(within(backlog).getByTestId("split-run-phase-artifacts-backlog")).toBeInTheDocument();
    expect(within(backlog).getByRole("button", { name: "description.md" })).toBeInTheDocument();
  });

  it("highlights a log line when it is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-stream-line-onrun-implement")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-onrun-implement")).toHaveAttribute("data-highlighted", "true");
    expect(screen.queryByTestId("split-run-canvas-node-onrun-implement")).not.toBeInTheDocument();
  });

  it("puts risk score and code quality on the verify step", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-impl",
                  step: "Implement",
                  stepIndex: 0,
                  state: "STATE_FINISHED",
                  result: "RESULT_PASSED",
                },
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 1,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    const verifyChecks = screen.getByTestId("split-run-phase-checks-verify-1");
    const risk = within(verifyChecks).getByRole("button", { name: /Risk score/ });
    expect(risk).toHaveTextContent("Risk score");
    expect(risk.className).toContain("bg-amber-500/10");
    expect(risk.className).not.toContain("bg-red-700");
    expect(within(verifyChecks).getByText("Code quality")).toBeInTheDocument();
    expect(within(verifyChecks).queryByText("Test coverage")).not.toBeInTheDocument();
    expect(within(verifyChecks).queryByText("Confidence score")).not.toBeInTheDocument();
    expect(within(verifyChecks).queryByText("CI")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
  });

  it("pins the pull request review to the waiting implement log", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Ship idempotent refund retries",
        lineDispatches: [
          {
            id: "dispatch-waiting",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    expect(screen.getByTestId("split-run-review")).toBeInTheDocument();
  });

  it("opens a compact check in the analysis dialog", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 2,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    await user.click(screen.getByTestId("split-run-check-check-risk-review"));

    expect(screen.getByRole("heading", { name: "Risk score" })).toBeInTheDocument();
    expect(screen.getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
  });

  it("keeps the state bar when logs are complete and the order waits with no note", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "dasdas",
        statusNotes: [],
        assignees: [{ id: "user-1", name: "test test" }],
        lineDispatches: [
          {
            id: "dispatch-wait",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-1",
                step: "dasdasdas",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByText("This task needs attention from test test.")).not.toBeInTheDocument();
    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "This task needs a decision" })).toBeInTheDocument();
    expect(within(note).getByText("Every automation finished. This task is ready to complete.")).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });

  it("keeps the running log visible when the task has no note or checks", () => {
    renderPopup({ fixture: { ...SPLIT_RUN_RUNNING, waitingNotes: [], checks: [] } });

    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Automations" })).not.toBeInTheDocument();
    expect(screen.queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-log-scroll")).toBeInTheDocument();
  });

  it("shows pull request review guidance beside the pull request list", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER) });

    const sidebar = screen.getByTestId("split-run-overview-sidebar");
    const note = within(sidebar).getByTestId("split-run-attention-note");
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
    expect(within(note).queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(within(note).queryByRole("list")).not.toBeInTheDocument();
    expect(note).toHaveTextContent("This task closes when the pull request is merged or closed.");
    const heading = within(note).getByRole("heading", { name: "The pull request is ready for review" });
    const reviewLink = within(note).getByRole("link", { name: "Review PR #6812" });
    expect(reviewLink).toHaveAttribute("href", "https://github.com/superplanehq/superplane/pull/6812");
    expect(reviewLink).not.toHaveClass("w-full");
    const closing = within(note).getByText("This task closes when the pull request is merged or closed.");
    expect(closing.parentElement).toBe(heading.parentElement);
    expect(note.querySelector(".lucide-git-pull-request")).toBeNull();
    expect(within(note).queryByText("PR Closure")).not.toBeInTheDocument();
    expect(within(note).queryByText(/ago/)).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: /Update manually/ })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "More actions" })).not.toBeInTheDocument();

    const moreActions = screen.getByRole("button", { name: "More actions" });
    expect(moreActions.closest("header")).not.toBeNull();
    await user.click(moreActions);
    const menu = await screen.findByRole("menu");
    expect(within(menu).getByRole("menuitem", { name: "Reject" })).toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: "Approve" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
  });

  it("enables merge on the pull request review strip when mergeability is true", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_OPEN") });

    const note = within(screen.getByTestId("split-run-overview-sidebar")).getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("split-run-merge-button")).toBeEnabled();
    await user.click(within(note).getByTestId("split-run-merge-method"));
    const menu = await screen.findByRole("menu");
    expect(
      within(menu)
        .getAllByRole("menuitem")
        .map((item) => item.textContent),
    ).toEqual(["✓ Squash and merge", "Rebase and merge"]);
    await user.keyboard("{Escape}");
    await user.click(within(note).getByTestId("split-run-merge-button"));
    expect(mergeMutate).toHaveBeenCalledTimes(1);
    expect(mergeMutate.mock.calls[0]?.[0]).toEqual({
      pullRequestId: "pr-open",
      mergeMethod: "MERGE_METHOD_SQUASH",
      expectedHeadSha: "abc123",
    });
  });

  it("disables merge and shows the reason on hover when automation is running", async () => {
    const user = userEvent.setup();
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_ACTIVE_RUN",
      message: "Automation is still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_OPEN") });

    const note = within(screen.getByTestId("split-run-overview-sidebar")).getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(within(note).getByTestId("split-run-merge-reason"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Automation is still running.");
  });

  it("hides merge when the pull request is merged", () => {
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_MERGED") });

    const note = within(screen.getByTestId("split-run-overview-sidebar")).getByTestId("split-run-attention-note");
    expect(within(note).queryByTestId("split-run-merge-button")).not.toBeInTheDocument();
    expect(within(note).getByTestId("split-run-pr-merged")).toHaveTextContent("The pull request is merged.");
  });

  it("hides work-order close actions when the user cannot update the task", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      canUpdate: false,
    });

    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stop")).not.toBeInTheDocument();
  });

  it("offers automation Stop on a live running task", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-running",
      fixture: SPLIT_RUN_RUNNING,
    });

    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
  });

  it("hides automation Stop when the user cannot update the task", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-running",
      fixture: SPLIT_RUN_RUNNING,
      canUpdate: false,
    });

    expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
  });

  it("hides automation Rerun when the failed phase has no step index", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-failed-implement",
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-failed-implement",
        title: "Later step",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-plan", step: "Planning", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
              { id: "e-impl", step: "Implement", state: "STATE_FINISHED", result: "RESULT_FAILED" },
              { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.queryByTestId(/split-run-phase-rerun-/)).not.toBeInTheDocument();
  });

  it("pins a default failed note and keeps Reopen on the note", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_FAILED_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "This task is closed as failed" })).toBeInTheDocument();
    expect(within(note).getByText("Reopen this task to start the line again.")).toBeInTheDocument();
    expect(within(note).queryByRole("link", { name: "Debug" })).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reopen" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
  });

  it("offers Reject and Rerun on a failed open implement", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-2",
        title: "Test task 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              { id: "e-plan", step: "Planning", state: "STATE_FINISHED", result: "RESULT_PASSED" },
              {
                id: "e-impl",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
              },
            ],
          },
        ],
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rerun step" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Choose how to stop" })).not.toBeInTheDocument();
  });

  it("offers Reject and Rerun after a person stops the run", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-stopped",
        title: "Stopped job",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_CANCELLED",
              },
            ],
          },
        ],
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "A person stopped this automation" })).toBeInTheDocument();
    expect(
      within(note).getByText("This automation did not finish. This task still needs a decision."),
    ).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(within(note).queryByRole("link", { name: "Debug" })).not.toBeInTheDocument();
    expect(screen.queryByTestId(/split-run-phase-rerun-/)).not.toBeInTheDocument();
  });

  it("names the person who stopped the automation", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          id: "wo-stopped-named",
          title: "Stopped job",
          state: "STATE_OPEN",
          lineDispatches: [
            {
              id: "d-1",
              line: { id: "line-1", name: "Software delivery" },
              state: "STATE_FINISHED",
              stepExecutions: [
                {
                  id: "e-impl",
                  step: "Implement",
                  stepIndex: 0,
                  state: "STATE_FINISHED",
                  result: "RESULT_CANCELLED",
                },
              ],
            },
          ],
        },
        { stoppedBy: { id: "user-1", name: "Alex", initials: "A" } },
      ),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("work-order-mention")).toHaveTextContent("Alex");
    expect(within(note).getByRole("heading", { name: /stopped this automation/ })).toBeInTheDocument();
  });

  it("names the person who marked the task successful, with avatar", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, {
        closer: {
          actor: { id: "user-1", name: "Alex", initials: "A", avatarUrl: "https://example.com/alex.png" },
        },
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("work-order-mention")).toHaveTextContent("Alex");
    expect(within(note).getByTestId("work-order-mention").querySelector("img")).toHaveAttribute(
      "src",
      "https://example.com/alex.png",
    );
    expect(within(note).getByRole("heading", { name: /marked this task as successful/ })).toBeInTheDocument();
  });

  it("keeps Reject and Approve off the header on a running order", () => {
    renderSplitRun();

    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stop")).not.toBeInTheDocument();
  });

  it("opens a draft refine view without Automations", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-work-order-tab")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-overview-sidebar")).not.toBeInTheDocument();
    const description = screen.getByTestId("split-run-description");
    expect(description).toHaveClass("justify-end");
    expect(within(description).getByTestId("split-run-source")).toHaveTextContent("Leonardo DiCaprio");
    expect(within(description).queryByText("Created manually")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-log-tab-dot")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(screen.queryByTestId("split-run-intent-composer-score-copy")).not.toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: /^Model/ })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-verdict-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("split-run-intent-plan-chip-analyzing")).not.toBeInTheDocument();
    const archive = screen.getByTestId("popup-work-order-archive-button");
    const share = screen.getByTestId("popup-work-order-copy-link-button");
    expect(archive).toHaveAttribute("aria-label", "Archive");
    expect(archive.compareDocumentPosition(share) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("work-order-split-run").className).toContain("w-[min(70rem");
    expect(screen.getByTestId("work-order-split-run").className).toContain(
      "has-[[data-refine-chat-solo]]:w-[min(48rem",
    );
    expect(screen.getByTestId("work-order-split-run").className).toContain(
      "has-[[data-refine-plan-open]]:w-[min(80rem",
    );
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    expect(screen.queryByRole("button", { name: "Plan" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-log-pane")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("hides Refine on a backlog draft", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.queryByRole("button", { name: "Refine" })).toBeNull();
  });

  it("tells a draft is under analysis and keeps Archive in the header", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
        analysisRuns: [
          {
            canvasId: "canvas-backlog",
            workOrderId: DRAFT_WORK_ORDER.id ?? "",
            run: {
              id: "run-analysis",
              canvasId: "canvas-backlog",
              state: "STATE_STARTED",
              createdAt: "2026-08-28T12:00:00Z",
              updatedAt: "2026-08-28T12:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(screen.queryByTestId("split-run-intent-composer-score-copy")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-verdict-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("split-run-intent-plan-chip-analyzing")).not.toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: /^Model/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
    expect(screen.getByTestId("popup-work-order-archive-button")).toHaveAttribute("aria-label", "Archive");
  });

  it("replaces chat with source, artifacts, and pull requests after Start", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER) });

    const request = screen.getByTestId("split-run-intent-request");
    const result = screen.getByTestId("split-run-intent-result");
    expect(within(request).getByTestId("split-run-overview-sidebar")).toBeInTheDocument();
    expect(within(request).getByRole("heading", { name: "Source" })).toBeInTheDocument();
    expect(within(request).getByRole("heading", { name: "Artifacts" })).toBeInTheDocument();
    expect(within(request).getByRole("heading", { name: "Pull requests" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(within(request).queryByTestId("split-run-overview-checks")).toBeNull();
    expect(within(request).queryByTestId("split-run-intent-chat")).toBeNull();
    expect(within(result).getByTestId("split-run-intent-summary")).toBeInTheDocument();
  });

  it("keeps source and spec after reopen when the started card has a score", async () => {
    const user = userEvent.setup();
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: OPEN_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks: OPEN_WORK_ORDER_CHECKS }),
    });

    await user.click(screen.getByRole("tab", { name: "Task" }));
    const request = screen.getByTestId("split-run-intent-request");
    expect(within(request).getByTestId("split-run-overview-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-result")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-overview-checks")).not.toBeInTheDocument();
  });

  it("hides source, artifacts, and pull requests on the description tab", async () => {
    renderPopup({
      fixture: withPublishedSpec(
        splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
      ),
    });

    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).queryByTestId("split-run-intent-session")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-description")).toHaveTextContent(
      "Webhook delivery stops after a transient provider error",
    );
    expect(within(tab).queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(within(tab).queryByTestId("split-run-overview-sidebar")).not.toBeInTheDocument();
    expect(within(tab).queryByRole("heading", { name: "Source" })).not.toBeInTheDocument();
    expect(within(tab).queryByRole("heading", { name: "Artifacts" })).not.toBeInTheDocument();
    expect(within(tab).queryByRole("heading", { name: "Pull requests" })).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-document")).toBeInTheDocument();
    expect(within(tab).queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
    expect(within(tab).queryByTestId("split-run-check-comment-wo-review-pay-842-confidence")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-draft-action-group")).getByRole("button", { name: "Start" }),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-attention-note")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    await userEvent.click(screen.getByRole("button", { name: "Plan" }));
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(false);
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    expect(within(tab).getByTestId("split-run-check-comment-check-risk-review")).not.toHaveAttribute("open");
    expect(within(tab).getByTestId("split-run-check-comment-check-code-coverage")).not.toHaveAttribute("open");
    expect(within(tab).getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
    expect(within(tab).getByText(/The change replaces the retry policy/)).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
  });

  it("keeps the verdict and both score slots on the matrix while a review draft is analyzed", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]) });

    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
    const verdict = within(tab).getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "analyzing");
    expect(within(verdict).getByTestId("split-run-intent-verdict-analyzing").querySelector(".t-matrix")).not.toBeNull();
    const chips = within(tab).getByTestId("split-run-intent-composer-chips");
    expect(within(chips).getByTestId("split-run-intent-composer-score-analyzing")).toBeInTheDocument();
    expect(within(chips).getByTestId("split-run-intent-composer-confidence-analyzing")).toBeInTheDocument();
    expect(within(chips).queryByTestId("split-run-intent-composer-confidence")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-status-card")).toHaveAttribute("data-slot", "frame");
    expect(within(tab).getByTestId("split-run-intent-document")).toBeInTheDocument();
    expect(within(tab).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(tab).getByRole("button", { name: /^Model/ })).toBeInTheDocument();
  });

  it("opens and closes a check with details and summary", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: withPublishedSpec(
        splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
      ),
    });

    await user.click(screen.getByRole("button", { name: "Plan" }));
    const risk = screen.getByTestId("split-run-check-comment-check-risk-review");
    const coverage = screen.getByTestId("split-run-check-comment-check-code-coverage");
    expect(risk).not.toHaveAttribute("open");
    expect(coverage).not.toHaveAttribute("open");

    await user.click(screen.getByTestId("split-run-check-comment-toggle-check-code-coverage"));
    expect(coverage).toHaveAttribute("open");
  });

  it("shows the full check summary without a one-line clamp", async () => {
    renderPopup({
      fixture: withPublishedSpec(
        splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
      ),
    });

    await userEvent.click(screen.getByRole("button", { name: "Plan" }));
    const risk = screen.getByTestId("split-run-check-comment-check-risk-review");
    const summary = within(risk).getByText(/Moderate risk: retry policy changes affect every refund path/);
    expect(summary.tagName).toBe("P");
    expect(summary).not.toHaveClass("truncate");
    expect(summary).toHaveClass("break-words");
  });

  it("keeps description checks collapsed when the task is not a draft", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 2,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    await user.click(screen.getByRole("tab", { name: "Task" }));
    expect(screen.getByTestId("split-run-check-comment-check-risk-review")).not.toHaveAttribute("open");
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
  });

  it("hides confidence on a downstream Description tab and keeps other checks", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_ENUM_ORDER, { checks: VERIFY_STEP_CHECKS }),
    });

    await user.click(screen.getByRole("tab", { name: "Task" }));
    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(within(tab).queryByText(/fit for an agent on this factory line/)).toBeNull();
    expect(within(tab).getByText("Risk score")).toBeInTheDocument();
  });

  it("shows the Ingest log when a GitHub automation created the draft", async () => {
    const user = userEvent.setup();
    renderPopup({
      factoryId: PRIMARY_FACTORY_ID,
      orderId: INGEST_DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER),
    });

    await openLogTab(user);
    expect(screen.getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-phase-backlog")).getAllByRole("button", { name: "description.md" }).length,
    ).toBeGreaterThan(0);
  });

  it("shows a successful result only in the Task tab sidebar", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER) });

    const sidebar = screen.getByTestId("split-run-overview-sidebar");
    const note = within(sidebar).getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByRole("heading", { name: "This task succeeded" })).toBeInTheDocument();
    expect(within(note).getByText("The work is done. The result met the goal.")).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reopen" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Task" })).toHaveAttribute("data-state", "active");

    const request = screen.getByTestId("split-run-intent-request");
    expect(within(request).getByTestId("split-run-overview-sidebar")).toBeInTheDocument();
    expect(within(request).getByRole("heading", { name: "Source" })).toBeInTheDocument();
    expect(within(request).getByRole("heading", { name: "Artifacts" })).toBeInTheDocument();
    expect(within(request).getByRole("heading", { name: "Pull requests" })).toBeInTheDocument();
    expect(within(request).queryByTestId("split-run-overview-checks")).toBeNull();
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();

    await openLogTab(user);
    expect(screen.queryByTestId("split-run-attention-note")).not.toBeInTheDocument();
  });

  it("shows an unsuccessful result only in the Task tab sidebar", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(BOARD_DONE_REJECTED_ORDER) });

    const sidebar = screen.getByTestId("split-run-overview-sidebar");
    const note = within(sidebar).getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "This task did not succeed" })).toBeInTheDocument();
    expect(within(note).getByText("The work is done. The result did not meet the goal.")).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reopen" })).not.toBeInTheDocument();

    await openLogTab(user);
    expect(screen.queryByTestId("split-run-attention-note")).not.toBeInTheDocument();
  });

  it("hides invented files and ledger pull requests on a live task", async () => {
    const user = userEvent.setup();
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: LINE_BOARD_DONE_RECEIPTS_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, { demoArtifacts: false }),
    });

    expect(screen.getByTestId("split-run-overview-sidebar")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /#510/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "closure.md" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /merge-screenshot/ })).not.toBeInTheDocument();

    await openLogTab(user);
    expect(screen.queryByRole("link", { name: /merge-screenshot/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /#510/ })).not.toBeInTheDocument();
  });

  it("shows a failed implement stream with Reject and Rerun on the note", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(FAILED_WORK_ORDER) });

    await openLogTab(user);
    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-implement-0")).toBeInTheDocument();
  });

  it("expands the new implement run after a rerun of the same step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      {
        id: "wo-2",
        title: "Test task 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              {
                id: "e-plan",
                step: "Planning",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                run: { id: "run-plan" },
              },
              {
                id: "e-impl-old",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                run: { id: "run-old" },
              },
              {
                id: "e-impl-new",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_STARTED",
                result: "RESULT_UNKNOWN",
                run: { id: "run-new" },
              },
            ],
          },
        ],
      },
      { lineId: "line-1" },
    );
    const implementPhases = fixture.phases.filter((phase) => phase.name === "Implement");

    renderPopup({ fixture });

    expect(screen.getByTestId(`split-run-phase-${implementPhases[1].id}`)).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId(`split-run-phase-${implementPhases[0].id}`)).not.toHaveAttribute("aria-current");
  });

  it("opens the selected step log when a log row is clicked", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    await user.click(within(screen.getByTestId("split-run-phase-backlog")).getByRole("button", { name: /^Backlog/ }));

    expect(screen.getByTestId("split-run-stream-backlog")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-backlog")).queryByText("Started")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-backlog")).getByText("On Issue Label")).toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
  });

  it("puts View Automation Run on an expanded log row", async () => {
    const user = userEvent.setup();
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    await openLogTab(user);

    const prCreation = screen.getByTestId("split-run-phase-pr-creation-2");
    const view = within(prCreation).getByRole("link", { name: "View run" });
    expect(view).toHaveAttribute(
      "href",
      factoryAppSplitRunPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "app-pr-closure", {
        from: "task",
        orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
        canvas: "closure",
      }),
    );
    expect(within(prCreation).queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();

    const backlog = screen.getByTestId("split-run-phase-backlog");
    expect(within(backlog).queryByRole("link", { name: "View run" })).not.toBeInTheDocument();
    await user.click(within(backlog).getByRole("button", { name: /^Backlog/ }));
    expect(within(backlog).queryByRole("link", { name: "View run" })).not.toBeInTheDocument();
    expect(within(backlog).queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("opens a mapped implement-running task on the implement log", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Implement job",
        lineDispatches: [
          {
            id: "dispatch-1",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.getByRole("heading", { name: "Implement job" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-ingest")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-analyze")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-plan")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-score")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-implement-0")).toBeInTheDocument();
    expect(screen.getAllByText("Implementation").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
  });

  it("highlights the PR Closure log for the selected canvas component", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        title: "Send refund receipts after provider confirm",
        lineStatus: "passed",
        footer: buildSplitRunFooter({ kind: "done" }),
        footerTone: "done",
        currentPhaseId: "done",
        phases: [
          ...SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" as const })),
          {
            id: "done",
            name: "Done",
            status: "passed",
            duration: "1m 12s",
            componentName: "PR Closure",
            artifacts: [],
            stream: [],
            canvasSteps: [],
          },
        ],
      },
    });

    await openLogTab(user);
    expect(screen.queryByTestId("split-run-stream-done")).not.toBeInTheDocument();
    await user.click(within(screen.getByTestId("split-run-phase-done")).getByRole("button", { name: /^Done/ }));

    const stream = screen.getByTestId("split-run-stream-done");
    expect(within(stream).queryByText("Started")).not.toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /merge-screenshot/ })).toBeInTheDocument();
    expect(within(stream).getByRole("link", { name: /#510/ })).toBeInTheDocument();

    await user.click(within(screen.getByTestId("split-run-stream-line-find-pull-request")).getByRole("button"));

    expect(screen.getByTestId("split-run-stream-line-find-pull-request")).toHaveAttribute("data-highlighted", "true");
    expect(screen.queryByTestId("split-run-canvas-node-find-pull-request")).not.toBeInTheDocument();
  });

  it("lets you rename the title on a draft", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    await user.click(screen.getByTestId("popup-work-order-title"));
    const titleInput = await screen.findByTestId("popup-work-order-title-input");
    await user.clear(titleInput);
    await user.type(titleInput, "Renamed draft");
    await user.keyboard("{Enter}");
    expect(screen.getByTestId("popup-work-order-title")).toHaveTextContent("Renamed draft");
  });

  it("does not let you edit a completed task", () => {
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        footer: buildSplitRunFooter({ kind: "done" }),
        footerTone: "done",
      },
    });

    expect(screen.queryByTestId("popup-work-order-title")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
  });
});
