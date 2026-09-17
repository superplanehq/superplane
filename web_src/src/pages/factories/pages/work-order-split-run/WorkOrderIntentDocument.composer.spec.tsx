import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_READINESS_NOTES } from "../../lib/draftReadiness";
import { CONFIDENCE_ANALYZING_TOOLTIP } from "../../workOrders/ConfidenceMeter";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import {
  analysisChat,
  HIGH_CONFIDENCE,
  INTENT,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  composerPng,
  renderIntentDocument,
  uploadedComposerImage,
  WAITING_COMPOSER_VIEW,
} from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
import { REFINE_LAYOUT_STORAGE_KEY } from "./refineLayoutPreference";
import { resetStreamMemoryForTests } from "./useStreamOnUpdate";

const { showErrorToast } = vi.hoisted(() => ({ showErrorToast: vi.fn() }));

vi.mock("@/lib/toast", () => ({ showErrorToast }));

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined, name?: string) =>
      id ? { id, name: name ?? "Ada Lovelace", initials: "AL" } : null,
    isLoading: false,
  }),
}));

describe("WorkOrderIntentDocument composer", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
    showErrorToast.mockReset();
  });

  it("keeps the latest plan sticky and toggles the spec column", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
        resultFooter={<div data-testid="split-run-review">Ready</div>}
        analysis={analysisChat({
          view: {
            machineStatus: "waiting",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [
              { id: "agent-1", kind: "text", role: "agent", text: "I published the spec." },
              { id: "plan-1", kind: "plan", role: "plan", score: 4 },
            ],
          },
        })}
      />,
    );

    const strip = screen.getByTestId("split-run-intent-plan-updated");
    expect(
      within(screen.getByTestId("split-run-intent-transcript")).queryByTestId("split-run-intent-plan-updated"),
    ).toBeNull();
    const chips = screen.getByTestId("split-run-intent-composer-chips");
    const showPlan = within(chips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.plan });
    expect(within(chips).getByTestId("split-run-intent-composer-score")).toHaveAccessibleName("Clarity 4/5");
    expect(within(chips).queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
    expect(showPlan).toHaveTextContent(CREATE_WITH_AGENT_COPY.plan);
    expect(showPlan).not.toHaveTextContent(CREATE_WITH_AGENT_COPY.showPlan);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    expect(within(strip).getByTestId("split-run-review")).toHaveTextContent("Ready");
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-decision")).not.toBeInTheDocument();

    await user.click(showPlan);
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({ planOpen: true });
    expect(screen.getByTestId("split-run-intent-result")).toBeInTheDocument();
    for (const column of screen.getAllByTestId("split-run-intent-chat-column")) {
      expect(column).toHaveClass("px-4");
    }
    expect(screen.getByTestId("split-run-intent-request").style.getPropertyValue("--intent-left")).toBe("50%");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(false);
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    const hidePlan = within(screen.getByTestId("split-run-intent-composer-chips")).getByRole("button", {
      name: CREATE_WITH_AGENT_COPY.plan,
    });
    expect(hidePlan).toHaveAttribute("aria-expanded", "true");
    expect(hidePlan).toHaveAttribute("aria-pressed", "true");
    expect(hidePlan).toHaveTextContent(CREATE_WITH_AGENT_COPY.plan);
    expect(within(hidePlan).queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-intent-result")).queryByRole("button", {
        name: CREATE_WITH_AGENT_COPY.plan,
      }),
    ).toBeNull();
    expect(screen.queryByTestId("split-run-intent-decision")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-intent-plan-updated")).getByTestId("split-run-review"),
    ).toHaveTextContent("Ready");

    await user.click(hidePlan);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(
      within(screen.getByTestId("split-run-intent-composer-chips")).getByRole("button", {
        name: CREATE_WITH_AGENT_COPY.plan,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(false);
  });

  it("keeps draft actions on the chips before a score exists", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        resultFooter={<div data-testid="split-run-review">Ready</div>}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    const card = screen.getByTestId("split-run-intent-status-card");
    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(card).toContainElement(chips);
    expect(screen.queryByTestId("split-run-intent-summary-drawer")).not.toBeInTheDocument();
    expect(within(chips).getByRole("status", { name: /^Clarity\./ })).toBeInTheDocument();
    expect(within(chips).getByRole("status", { name: /^Confidence\./ })).toBeInTheDocument();
    expect(within(card).getByTestId("split-run-review")).toHaveTextContent("Ready");
    expect(within(card).getByTestId("split-run-intent-verdict")).toHaveAttribute("data-tone", "analyzing");
  });

  it("hides the plan toggle until a spec exists", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(chips).queryByRole("button", { name: CREATE_WITH_AGENT_COPY.plan })).not.toBeInTheDocument();
    const analyzing = screen.getByTestId("split-run-intent-verdict-analyzing");
    expect(analyzing.querySelector(".t-matrix")).not.toBeNull();
    expect(analyzing).not.toHaveTextContent("Analyzing");
    expect(screen.getByTestId("split-run-intent-verdict")).toHaveTextContent(DRAFT_READINESS_NOTES.analyzing.headline);
    expect(screen.queryByTestId("split-run-intent-composer-score")).not.toBeInTheDocument();
  });

  it("stops the matrix when analysis ends without a score", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        analysis={analysisChat({
          view: {
            machineStatus: "failed",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    expect(screen.getByTestId("split-run-intent-composer-score")).toHaveTextContent("–");
    expect(screen.getByTestId("split-run-intent-composer-confidence")).toHaveTextContent("–");
    expect(screen.queryByTestId("split-run-intent-verdict-analyzing")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-verdict")).toHaveAttribute("data-tone", "pending");
  });

  it("marks the Plan toggle when the spec changes", () => {
    const { rerender } = renderIntentDocument(
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
            messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 4 }],
          },
        })}
      />,
    );

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(chips).queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();

    rerender(
      <TooltipProvider>
        <WorkOrderIntentDocument
          {...INTENT_DOC}
          artifacts={[{ ...INTENT, data: { ...INTENT.data, body: `${INTENT.data.body}\n\nMore scope.\n` } }]}
          confidence={HIGH_CONFIDENCE}
          analysis={analysisChat({
            view: {
              machineStatus: "waiting",
              canvasId: "canvas-1",
              canvasRunId: "run-1",
              executionId: "exec-1",
              messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 4 }],
            },
          })}
        />
      </TooltipProvider>,
    );

    const updated = within(screen.getByTestId("split-run-intent-composer-chips")).getByTestId(
      "split-run-intent-plan-status",
    );
    expect(updated).toHaveAccessibleName(CREATE_WITH_AGENT_COPY.planUpdated);
    expect(updated).toHaveAttribute("role", "status");
    expect(within(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.plan })).getByRole("status")).toBe(updated);
  });

  it("explains the score matrix on hover while the agent works", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    await user.hover(screen.getByRole("status", { name: /^Clarity\./ }));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(CONFIDENCE_ANALYZING_TOOLTIP);
  });

  it("replaces the scores and the plan badge with the matrix while the agent works", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={{ ...HIGH_CONFIDENCE, score: 2 }}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 2 }],
          },
        })}
      />,
    );

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(chips).getByRole("status", { name: /^Clarity\./ })).toBeInTheDocument();
    expect(within(chips).getByRole("status", { name: /^Confidence\./ })).toBeInTheDocument();
    const cardMatrix = screen.getByTestId("split-run-intent-verdict-analyzing");
    expect(cardMatrix.querySelector(".t-matrix")).not.toBeNull();
    expect(cardMatrix).not.toHaveTextContent("Analyzing");
    const planMatrix = within(chips).getByTestId("split-run-intent-plan-chip-analyzing");
    expect(planMatrix.querySelector(".t-matrix")).not.toBeNull();
    expect(planMatrix).not.toHaveTextContent("Analyzing");
    expect(within(chips).queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer-score")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-verdict")).toHaveAttribute("data-tone", "analyzing");
  });

  it("attaches, pastes, caps, and sends composer images", async () => {
    const user = userEvent.setup();
    let nextId = 0;
    const onUploadFiles = vi.fn(async (files: FileList | File[]) =>
      Array.from(files).map((file) => uploadedComposerImage(`file-${++nextId}`, file.name)),
    );
    const onSend = vi.fn();
    const input = () => screen.getByTestId("create-work-order-request-image-input");
    const { rerender } = renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({ view: WAITING_COMPOSER_VIEW, onUploadFiles, onSend })}
      />,
    );
    await user.upload(input(), new File(["notes"], "notes.md", { type: "text/markdown" }));
    expect(onUploadFiles).not.toHaveBeenCalled();
    await user.upload(input(), composerPng("bug.png"));
    await waitFor(() => expect(screen.getByTestId("create-work-order-request-attachment-file-1")).toBeInTheDocument());
    fireEvent.paste(screen.getByTestId("split-run-intent-composer"), {
      clipboardData: { files: [composerPng("shot.png")], getData: () => "" },
    });
    await waitFor(() => expect(screen.getByTestId("create-work-order-request-attachment-file-2")).toBeInTheDocument());
    await user.click(screen.getByTestId("create-work-order-request-attachment-file-2"));
    await user.click(screen.getByTestId("create-work-order-request-image-remove"));
    await user.upload(
      input(),
      Array.from({ length: 10 }, (_, index) => composerPng(`extra-${index}.png`)),
    );
    await waitFor(() => expect(screen.getAllByTestId(/create-work-order-request-attachment-/)).toHaveLength(8));
    expect(showErrorToast).toHaveBeenCalledWith("Attachments are limited to 8 images.");
    await user.click(screen.getByTestId("split-run-intent-composer-send"));
    const uploaded = uploadedComposerImage("file-1", "bug.png");
    rerender(
      <TooltipProvider>
        <WorkOrderIntentDocument
          {...INTENT_DOC}
          artifacts={[INTENT]}
          files={[{ id: uploaded.id, downloadUrl: uploaded.previewUrl }]}
          analysis={analysisChat({
            view: {
              ...WAITING_COMPOSER_VIEW,
              messages: [{ id: "user-1", kind: "text", role: "user", text: String(onSend.mock.calls[0]?.[0]) }],
            },
          })}
        />
      </TooltipProvider>,
    );
    expect(
      within(screen.getByTestId("split-run-intent-user-note")).getByRole("img", { name: "bug.png" }),
    ).toHaveAttribute("src", uploaded.previewUrl);
    expect(screen.getByTestId("split-run-description")).not.toHaveTextContent("bug.png");
  });
});
