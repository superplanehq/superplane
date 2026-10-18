import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { SplitRunReview } from "./SplitRunReview";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import {
  analysisChat,
  HIGH_CLARITY,
  HIGH_CONFIDENCE,
  INTENT,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  renderIntentDocument,
} from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
import { REFINE_LAYOUT_STORAGE_KEY } from "./refineLayoutPreference";
import { resetStreamMemoryForTests } from "./useStreamOnUpdate";

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined, name?: string) =>
      id ? { id, name: name ?? "Ada Lovelace", initials: "AL" } : null,
    isLoading: false,
  }),
}));

const WAITING_WITH_PLAN = {
  machineStatus: "waiting" as const,
  canvasId: "canvas-1",
  canvasRunId: "run-1",
  executionId: "exec-1",
  messages: [{ id: "plan-1", kind: "plan" as const, role: "plan" as const, score: 4 }],
};

describe("WorkOrderIntentDocument score evidence", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
  });

  it.each([2, 4])("shows the review controls for a stored confidence %s", (score) => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={{ ...HIGH_CONFIDENCE, score }}
        analysis={analysisChat({
          view: {
            machineStatus: "waiting",
            messages: [{ id: "agent-1", kind: "text", role: "agent", text: "I need one detail." }],
          },
        })}
      />,
    );
    expect(screen.getByRole("button", { name: "Open plan" })).toBeVisible();
    expect(screen.getByRole("button", { name: `Confidence ${score}/5` })).toBeVisible();
  });

  it.each([
    { score: 2, clarity: 4 },
    { score: 4, clarity: 4 },
    { score: 4, clarity: 2 },
  ])("shows pending questions with confidence $score and clarity $clarity", async ({ score, clarity }) => {
    const user = userEvent.setup();
    const onSubmitSurvey = vi.fn();
    const onStart = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={{ ...HIGH_CLARITY, score: clarity }}
        confidence={{ ...HIGH_CONFIDENCE, score }}
        resultFooter={
          <SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} onStart={onStart} compact />
        }
        analysis={analysisChat({
          onSubmitSurvey,
          modelSelect: <button type="button">Model: Auto</button>,
          view: {
            machineStatus: "waiting",
            messages: [{ id: "agent-1", kind: "text", role: "agent", text: "I need one detail." }],
            survey: { id: "survey-1", questions: [{ prompt: "Which customers?", options: ["Existing customers"] }] },
          },
        })}
      />,
    );

    const plan = screen.getByRole("region", { name: "Plan" });
    expect(within(plan).getByRole("button", { name: `Confidence ${score}/5` })).toBeInTheDocument();
    expect(within(plan).getByRole("button", { name: "Open plan" })).toBeInTheDocument();
    expect(plan).toHaveTextContent("Clearer empty state");
    expect(
      plan.compareDocumentPosition(screen.getByText("Which customers?")) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "Tell the agent more about this task" })).not.toBeInTheDocument();
    const implementation = screen.getByRole("region", { name: "Implementation" });
    expect(within(implementation).getByRole("button", { name: "Model: Auto" })).toBeVisible();
    const start = within(implementation).getByRole("button", { name: "Start" });
    expect(start).toBeEnabled();
    if (score < 3 || clarity < 3) {
      expect(implementation).toHaveTextContent("Review the plan before you start");
    }
    expect(screen.queryByRole("button", { name: "Suggest changes" })).not.toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: /Which customers/ }), "Only paying customers");
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswer }));
    expect(onSubmitSurvey).toHaveBeenCalledWith(expect.stringContaining("Only paying customers"));
    await user.click(start);
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("keeps survey answers and errors visible without a plan, then restores the composer", async () => {
    const user = userEvent.setup();
    const analysis = analysisChat({
      composer: "Keep this draft message",
      view: {
        machineStatus: "waiting",
        messages: [{ id: "agent-1", kind: "text", role: "agent", text: "I need one detail." }],
        survey: { id: "survey-1", questions: [{ prompt: "Which customers?", options: ["Existing customers"] }] },
      },
    });
    const document = (next: typeof analysis) => (
      <TooltipProvider>
        <WorkOrderIntentDocument {...INTENT_DOC} artifacts={[]} analysis={next} />
      </TooltipProvider>
    );
    const { rerender } = renderIntentDocument(
      <WorkOrderIntentDocument {...INTENT_DOC} artifacts={[]} analysis={analysis} />,
    );
    expect(screen.queryByRole("region", { name: "Plan" })).not.toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Implementation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open plan" })).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Tell the agent more about this task" })).not.toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: /Which customers/ }), "Paying customers");

    rerender(document({ ...analysis, canSend: false }));
    expect(screen.getByRole("textbox", { name: /Which customers/ })).toBeDisabled();
    rerender(document({ ...analysis, composerError: "The answer did not send. Try again." }));
    expect(screen.getByRole("alert")).toHaveTextContent("The answer did not send. Try again.");
    expect(screen.getByRole("textbox", { name: /Which customers/ })).toHaveValue("Paying customers");
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswer })).toBeEnabled();

    rerender(
      document({
        ...analysis,
        view: {
          ...analysis.view,
          messages: [
            ...analysis.view.messages,
            { id: "reply-1", kind: "text", role: "user", text: "Paying customers" },
          ],
        },
      }),
    );
    expect(screen.queryByRole("textbox", { name: /Which customers/ })).not.toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Tell the agent more about this task" })).toHaveValue(
      "Keep this draft message",
    );
  });

  it("uses the plan title as the open action and keeps both scores as evidence", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    const verdict = within(card).getByTestId("split-run-intent-verdict");
    expect(within(verdict).getByRole("button", { name: "Open plan" })).toHaveTextContent("Clearer empty state");
    expect(verdict).not.toHaveTextContent("Draft plan");
    const clarity = within(card).getByTestId("split-run-intent-composer-score");
    const confidence = within(card).getByTestId("split-run-intent-composer-confidence");
    expect(clarity).toHaveAccessibleName("Clarity 4/5");
    expect(clarity).toHaveAttribute("aria-expanded", "false");
    expect(confidence).toHaveAccessibleName("Confidence 4/5");
    expect(
      within(clarity).getByTestId("split-run-intent-composer-score-meter").querySelectorAll("[data-filled='true']"),
    ).toHaveLength(4);
    expect(screen.queryByTestId("split-run-intent-composer-score-copy")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-summary-drawer")).not.toBeInTheDocument();
  });

  it("keeps ready Start below the visible composer", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={HIGH_CONFIDENCE}
        resultFooter={<SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} compact />}
        analysis={analysisChat({
          view: WAITING_WITH_PLAN,
          modelSelect: <button type="button">Model: Auto</button>,
        })}
      />,
    );

    const card = screen.getByTestId("split-run-intent-status-card");
    const verdict = within(card).getByTestId("split-run-intent-verdict");
    expect(within(verdict).getByRole("button", { name: "Open plan" })).toBeInTheDocument();
    const settings = screen.getByRole("region", { name: "Implementation" });
    expect(within(settings).getByText("Implementation")).toBeVisible();
    const plan = within(card).getByRole("button", { name: "Open plan" });
    const model = within(settings).getByRole("button", { name: "Model: Auto" });
    const start = within(settings).getByRole("button", { name: "Start" });
    expect(plan.compareDocumentPosition(model) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(model.compareDocumentPosition(start) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByTestId("split-run-intent-chat-log")).not.toContainElement(settings);
    expect(start).toHaveClass("bg-primary");
    expect(settings).not.toHaveTextContent("Starting not recommended");
    expect(within(settings).getByTestId("split-run-draft-action-group")).not.toHaveClass("border");
    expect(screen.queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
    const composer = screen.getByRole("textbox", { name: "Tell the agent more about this task" });
    expect(composer).toBeVisible();
    expect(within(settings).queryByRole("button", { name: "Suggest changes" })).not.toBeInTheDocument();
    expect(composer.compareDocumentPosition(settings) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByTestId("split-run-intent-chat-log")).not.toContainElement(
      screen.getByRole("region", { name: "Implementation" }),
    );
    expect(screen.getByRole("button", { name: "Start" })).toBeEnabled();
  });

  it.each([
    { name: "low clarity", clarity: 2 },
    { name: "active analysis", machineStatus: "running" as const },
    { name: "missing plan", noPlan: true },
    { name: "active analysis without a plan", machineStatus: "running" as const, noPlan: true },
    { name: "existing draft", composer: "Keep this draft" },
    { name: "send error", composerError: "The message did not send. Try again." },
  ])("keeps the composer visible with high confidence and $name", (state) => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={state.noPlan ? [] : [INTENT]}
        clarity={{ ...HIGH_CLARITY, score: state.clarity ?? 4 }}
        confidence={HIGH_CONFIDENCE}
        resultFooter={<SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} compact />}
        analysis={analysisChat({
          composer: state.composer ?? "",
          composerError: state.composerError,
          view: { ...WAITING_WITH_PLAN, machineStatus: state.machineStatus ?? "waiting", messages: [] },
        })}
      />,
    );
    expect(screen.getByRole("textbox", { name: "Tell the agent more about this task" })).toHaveValue(
      state.composer ?? "",
    );
    expect(screen.queryByRole("button", { name: "Suggest changes" })).not.toBeInTheDocument();
    if (state.noPlan) {
      expect(screen.queryByRole("region", { name: "Implementation" })).not.toBeInTheDocument();
    }
  });

  it.each([1, 2])("keeps secondary Start available with one warning at confidence %s", async (score) => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score }}
        resultFooter={
          <SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} onStart={onStart} compact />
        }
        analysis={analysisChat({ view: WAITING_WITH_PLAN, modelSelect: <button type="button">Model: Auto</button> })}
      />,
    );

    expect(onStart).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Model: Auto" })).toBeVisible();
    const start = screen.getByRole("button", { name: "Start" });
    expect(start).not.toHaveClass("bg-primary");
    expect(start).toHaveClass("border");
    expect(screen.getByTestId("split-run-intent-chat-log")).not.toContainElement(
      screen.getByRole("region", { name: "Implementation" }),
    );
    expect(screen.getByRole("region", { name: "Implementation" })).toHaveTextContent(
      "Review the plan before you start",
    );
    expect(start).toBeEnabled();
    expect(screen.queryByTestId("split-run-intent-settings")).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Tell the agent more about this task" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Suggest changes" })).not.toBeInTheDocument();
    await user.click(start);
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("peeks a score summary on hover and pins it on click", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score: 3, summary: "The change crosses billing and the API." }}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    expect(confidence).toHaveAccessibleName("Confidence 3/5");

    await user.hover(confidence);
    expect(await screen.findByTestId("split-run-intent-composer-confidence-copy")).toHaveTextContent(
      "The change crosses billing and the API.",
    );
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    expect(confidence).toHaveAttribute("aria-pressed", "false");

    await user.unhover(confidence);
    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-pressed", "true");
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    await user.unhover(confidence);
    expect(screen.getByTestId("split-run-intent-composer-confidence-copy")).toBeInTheDocument();

    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-pressed", "false");

    await user.hover(screen.getByTestId("split-run-intent-composer-score"));
    expect(await screen.findByTestId("split-run-intent-composer-score-copy")).toHaveTextContent(HIGH_CLARITY.summary);
    expect(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY)).toBeNull();
  });

  it("shows a dash for a score the analysis has not published", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({
          view: {
            machineStatus: "waiting",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    const clarity = screen.getByTestId("split-run-intent-composer-score");
    expect(clarity).toHaveTextContent("–");
    expect(clarity).not.toHaveAttribute("aria-expanded");
    expect(screen.getByTestId("split-run-intent-composer-confidence")).toHaveAccessibleName("Confidence 4/5");
  });

  it("restores the plan pane from localStorage and ignores legacy summary keys", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: "clarity", planOpen: true }));
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "open");
    expect(screen.queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
  });

  it("does not open the plan pane from storage before a spec exists", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ planOpen: true }));
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    expect(screen.queryByRole("button", { name: /^(Open|Hide) plan$/ })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(false);
  });
});

describe("WorkOrderIntentDocument narrow implementation controls", () => {
  const desktopWidth = window.innerWidth;

  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
    Object.defineProperty(window, "innerWidth", { configurable: true, writable: true, value: 390 });
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    Object.defineProperty(window, "innerWidth", { configurable: true, writable: true, value: desktopWidth });
    resetStreamMemoryForTests();
  });

  function renderDiscouragedDraft(onStart = vi.fn()) {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score: 3 }}
        resultFooter={
          <SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} onStart={onStart} compact />
        }
        analysis={analysisChat({ view: WAITING_WITH_PLAN, modelSelect: <button type="button">Model: Auto</button> })}
      />,
    );
    return onStart;
  }

  it("shows the plan warning and secondary Start without another step", async () => {
    const user = userEvent.setup();
    const onStart = renderDiscouragedDraft();

    expect(onStart).not.toHaveBeenCalled();
    const implementation = screen.getByRole("region", { name: "Implementation" });
    const headline = within(implementation).getByText("Review the plan before you start");
    expect(headline).toBeVisible();

    const actions = screen.getByTestId("split-run-intent-settings");
    expect(actions).toContainElement(screen.getByRole("button", { name: "Model: Auto" }));
    expect(actions).toContainElement(screen.getByRole("button", { name: "Start" }));
    const start = screen.getByRole("button", { name: "Start" });
    expect(start).not.toHaveClass("bg-primary");
    expect(start).toHaveClass("border");

    await user.click(start);
    expect(onStart).toHaveBeenCalledTimes(1);
  });
});
