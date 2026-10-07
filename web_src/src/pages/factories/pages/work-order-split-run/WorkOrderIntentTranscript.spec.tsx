import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage, CreateWithAgentView } from "../createWithAgentTypes";
import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";
import { WorkOrderIntentRequest } from "./WorkOrderIntentRequest";
import { WorkOrderIntentTranscript } from "./WorkOrderIntentTranscript";

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined) => {
      if (id === "user-ada") {
        return { id, name: "Ada Lovelace", initials: "AL", avatarUrl: "https://example.com/ada.png" };
      }
      if (id === "user-alan") {
        return { id, name: "Alan Turing", initials: "AT" };
      }
      return null;
    },
    isLoading: false,
  }),
}));

let notifyResize: () => void;

class MockResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    notifyResize = () => callback([], this as unknown as ResizeObserver);
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}

function renderTranscript(messages: CreateWithAgentMessage[], extras: { streaming?: boolean } = {}) {
  return render(<WorkOrderIntentTranscript organizationId="org-1" messages={messages} streaming={extras.streaming} />);
}

function expectAvatarBeforeGivenName(container: HTMLElement, avatarName: string, givenName: string) {
  const avatar = within(container).getByRole("img", { name: avatarName });
  const name = within(container).getByText(givenName);
  expect(avatar.compareDocumentPosition(name) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
}

describe("WorkOrderIntentTranscript", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not restream the last agent line after a user reply", () => {
    renderTranscript(
      [
        { id: "agent-1", kind: "text", role: "agent", text: "I asked one survey question." },
        { id: "user-1", kind: "text", role: "user", origin: "survey", text: "Add separate lists per animal." },
      ],
      { streaming: true },
    );

    const transcript = screen.getByTestId("split-run-intent-transcript");
    expect(transcript.querySelector(".sp-stream-w")).toBeNull();
    expect(transcript.querySelector(".sp-text-reveal")).not.toBeNull();
  });

  it("streams the latest agent line while that turn is still open", () => {
    renderTranscript([{ id: "agent-1", kind: "text", role: "agent", text: "I asked one survey question." }], {
      streaming: true,
    });

    expect(screen.getByTestId("split-run-intent-transcript").querySelectorAll(".sp-stream-w")).not.toHaveLength(0);
  });

  it("streams a final agent line that arrives as the run becomes idle", () => {
    const { rerender } = renderTranscript([]);

    rerender(
      <WorkOrderIntentTranscript
        organizationId="org-1"
        messages={[{ id: "agent-1", kind: "text", role: "agent", text: "The final answer is ready." }]}
      />,
    );

    const words = screen.getByTestId("split-run-intent-transcript").querySelectorAll(".sp-stream-w");
    expect(words).toHaveLength(5);
    expect(words[0]).toHaveClass("is-streaming");
  });

  it("continues a final message animation after the parent stops streaming", () => {
    const { rerender } = renderTranscript([]);
    const messages: CreateWithAgentMessage[] = [
      { id: "agent-1", kind: "text", role: "agent", text: "The final answer is ready." },
    ];

    rerender(<WorkOrderIntentTranscript organizationId="org-1" messages={messages} />);
    const transcript = screen.getByTestId("split-run-intent-transcript");
    const firstWord = transcript.querySelector(".sp-stream-w");

    rerender(<WorkOrderIntentTranscript organizationId="org-1" messages={messages} streaming={false} />);

    expect(transcript.querySelector(".sp-stream-w")).toBe(firstWord);
    expect(transcript.querySelectorAll(".sp-stream-w.is-streaming")).toHaveLength(5);
  });

  it("shows survey answers as question labels with primary pick bubbles", () => {
    renderTranscript([
      {
        id: "user-1",
        kind: "text",
        role: "user",
        origin: "survey",
        userId: "user-ada",
        text: "What should the agent build? Custom styled modal\nHow is the work done? Reviewer approves by taste",
      },
    ]);

    const answer = screen.getByTestId("split-run-intent-survey-answer");
    expect(answer).toHaveTextContent(CREATE_WITH_AGENT_COPY.answeredBy);
    expect(answer).toHaveTextContent("Ada");
    expectAvatarBeforeGivenName(answer, "Ada Lovelace", "Ada");
    expect(answer).toHaveTextContent("What should the agent build?");
    expect(answer).toHaveTextContent("Custom styled modal");
    expect(answer).toHaveTextContent("How is the work done?");
    expect(answer).toHaveTextContent("Reviewer approves by taste");
    const picks = within(answer).getAllByTestId("split-run-intent-survey-pick");
    expect(picks.map((pick) => pick.textContent)).toEqual(["Custom styled modal", "Reviewer approves by taste"]);
    for (const pick of picks) {
      expect(pick).toHaveClass("sp-chat-outgoing");
    }
    expect(answer).toHaveClass("items-end");
    expect(answer.parentElement).toHaveClass("justify-end", "pt-2.5", "pb-2.5");
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.youSurvey)).not.toBeInTheDocument();
  });

  it("shows composer notes with the sender given name and avatar", () => {
    renderTranscript([
      { id: "user-1", kind: "text", role: "user", userId: "user-ada", text: "Keep the current dark theme." },
    ]);

    const note = screen.getByTestId("split-run-intent-user-note");
    expect(note.querySelector(".sp-chat-outgoing")).not.toBeNull();
    expect(note.querySelector(".sp-chat-outgoing")).not.toHaveClass("border");
    expect(note).toHaveTextContent("Ada");
    expect(within(note).getByRole("img", { name: "Ada Lovelace" })).toHaveAttribute(
      "src",
      "https://example.com/ada.png",
    );
    expectAvatarBeforeGivenName(note, "Ada Lovelace", "Ada");
    expect(note).toHaveTextContent("Keep the current dark theme.");
    expect(note.parentElement).toHaveClass("justify-end", "pt-2.5", "pb-2.5");
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.you)).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
  });

  it("keeps a 12px gap for user-to-agent and user-to-user turns", () => {
    renderTranscript([
      { id: "agent-1", kind: "text", role: "agent", text: "I asked one survey question." },
      { id: "user-1", kind: "text", role: "user", origin: "survey", text: "Add separate lists per animal." },
      { id: "user-2", kind: "text", role: "user", text: "Keep the current dark theme." },
      { id: "agent-2", kind: "text", role: "agent", text: "I will keep that theme." },
    ]);

    const survey = screen.getByTestId("split-run-intent-survey-answer").parentElement;
    const note = screen.getByTestId("split-run-intent-user-note").parentElement;
    expect(survey).toHaveClass("pt-2.5", "pb-1.5");
    expect(note).toHaveClass("pt-1.5", "pb-2.5");
    for (const agent of screen.getAllByTestId("split-run-intent-agent-message")) {
      expect(agent).toHaveClass("py-0.5");
    }
  });

  it("shows two senders on the same session as different people", () => {
    renderTranscript([
      { id: "user-1", kind: "text", role: "user", userId: "user-ada", text: "Keep the current dark theme." },
      { id: "user-2", kind: "text", role: "user", userId: "user-alan", text: "Use the existing retry helper." },
    ]);

    const notes = screen.getAllByTestId("split-run-intent-user-note");
    expect(notes[0]).toHaveTextContent("Ada");
    expect(notes[1]).toHaveTextContent("Alan");
  });

  it("shows the sender once for consecutive turns from the same person", () => {
    renderTranscript([
      { id: "user-1", kind: "text", role: "user", userId: "user-ada", text: "Keep the current dark theme." },
      { id: "user-2", kind: "text", role: "user", origin: "survey", userId: "user-ada", text: "Scope? One file" },
      { id: "agent-1", kind: "text", role: "agent", text: "Noted." },
      { id: "user-3", kind: "text", role: "user", userId: "user-ada", text: "Also keep the retry helper." },
    ]);

    const notes = screen.getAllByTestId("split-run-intent-user-note");
    expect(notes[0]).toHaveTextContent("Ada");
    expect(screen.getByTestId("split-run-intent-survey-answer")).not.toHaveTextContent("Ada");
    expect(notes[1]).toHaveTextContent("Ada");
  });

  it("omits the sender header when the message has no user id", () => {
    renderTranscript([{ id: "user-1", kind: "text", role: "user", text: "Keep the current dark theme." }]);

    const note = screen.getByTestId("split-run-intent-user-note");
    expect(note).toHaveTextContent("Keep the current dark theme.");
    expect(note).not.toHaveTextContent("Ada");
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.you)).not.toBeInTheDocument();
  });

  it("keeps agent lines on the left with a little side padding", () => {
    render(
      <WorkOrderIntentTranscript
        organizationId="org-1"
        messages={[
          { id: "agent-1", kind: "text", role: "agent", text: "I updated the plan.", activityId: "activity-1" },
        ]}
        activities={[
          {
            id: "activity-1",
            provider: "codex",
            status: "passed",
            sequence: 1,
            items: [
              {
                type: "tool",
                id: "command-1",
                kind: "bash",
                name: "Bash",
                input: "ls",
                output: "",
                outputStreams: [],
                status: "passed",
                truncated: false,
              },
            ],
            truncated: false,
          },
        ]}
      />,
    );

    const agent = screen.getByTestId("split-run-intent-agent-message");
    expect(agent).toHaveTextContent("I updated the plan.");
    expect(agent).toHaveClass("px-2");
    expect(agent.closest(".justify-end")).toBeNull();
    expect(screen.getByTestId("split-run-intent-transcript")).toHaveClass("space-y-0");
    expect(screen.getByTestId("agent-activity-activity-1")).toHaveClass("px-2");
    expect(screen.getByRole("button", { name: "Explored codebase" })).toBeInTheDocument();
  });

  it("collapses a long composer note and expands it on Show more", async () => {
    const user = userEvent.setup();
    renderTranscript([
      {
        id: "user-1",
        kind: "text",
        role: "user",
        text: "Need a payment method.\n".repeat(40),
      },
    ]);

    const note = screen.getByTestId("split-run-intent-user-note");
    const content = within(note).getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });
    act(() => notifyResize());

    expect(within(note).getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "220px" });
    expect(within(note).getByTestId("work-order-description").querySelector(".sp-chat-outgoing-fade")).not.toBeNull();

    await user.click(within(note).getByRole("button", { name: /show more/i }));
    expect(within(note).getByRole("button", { name: /show less/i })).toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "220px" });
  });

  it("uses the download URL for images in composer notes", () => {
    const fileId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    render(
      <WorkOrderIntentTranscript
        organizationId="org-1"
        files={[{ id: fileId, downloadUrl: "https://cdn.example/bug.png" }]}
        messages={[{ id: "user-1", kind: "text", role: "user", text: `See ![bug](sp-file://${fileId})` }]}
      />,
    );

    expect(screen.getByRole("img", { name: "bug" })).toHaveAttribute("src", "https://cdn.example/bug.png");
  });

  it("hides plan-updated rows so the sticky control can own the latest plan", () => {
    renderTranscript([
      { id: "agent-1", kind: "text", role: "agent", text: "I published the spec." },
      { id: "plan-1", kind: "plan", role: "plan", score: 4 },
    ]);

    expect(screen.queryByTestId("split-run-intent-plan-updated")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.planUpdated })).not.toBeInTheDocument();
    expect(screen.getByText("I published the spec.")).toBeInTheDocument();
  });

  it("shows a sent time on hover or focus and keeps the message text in place", () => {
    const sentAt = new Date(2026, 7, 6, 10, 17);
    const later = new Date(2026, 7, 6, 10, 18);
    renderTranscript([
      {
        id: "user-1",
        kind: "text",
        role: "user",
        userId: "user-ada",
        text: "Keep the theme.",
        createdAtMs: sentAt.getTime(),
      },
      {
        id: "user-2",
        kind: "text",
        role: "user",
        userId: "user-ada",
        text: "Also keep the helper.",
        createdAtMs: later.getTime(),
      },
      {
        id: "user-3",
        kind: "text",
        role: "user",
        origin: "survey",
        text: "Scope? One file",
        createdAtMs: sentAt.getTime(),
      },
      { id: "agent-1", kind: "text", role: "agent", text: "Noted.", createdAtMs: sentAt.getTime() },
    ]);

    const noteTimes = screen.getAllByTestId("split-run-intent-user-note").map((note) => within(note).getByRole("time"));
    expect(noteTimes).toHaveLength(2);
    expectHoverSentTime(noteTimes[0], sentAt, "beside");
    expectHoverSentTime(noteTimes[1], later, "beside");
    expect(screen.getAllByTestId("split-run-intent-user-note")[1]).toHaveTextContent("Also keep the helper.");

    const surveyTime = within(screen.getByTestId("split-run-intent-survey-answer")).getByRole("time");
    expectHoverSentTime(surveyTime, sentAt, "beside");

    const agent = screen.getByTestId("split-run-intent-agent-message");
    expectHoverSentTime(within(agent).getByRole("time"), sentAt, "overlay");
    expect(agent).toHaveTextContent("Noted.");
    agent.focus();
    expect(agent).toHaveFocus();
  });

  it("omits a sent time when the message time is missing", () => {
    renderTranscript([
      { id: "user-1", kind: "text", role: "user", text: "Keep the theme." },
      { id: "agent-1", kind: "text", role: "agent", text: "Noted." },
      { id: "user-2", kind: "text", role: "user", text: "Still no clock.", createdAtMs: Number.NaN },
    ]);

    expect(screen.queryByRole("time")).not.toBeInTheDocument();
    for (const note of screen.getAllByTestId("split-run-intent-user-note")) {
      expect(note.parentElement).not.toHaveAttribute("tabindex");
    }
  });

  it("does not label a plan card or a tool activity row", () => {
    const sentAt = new Date(2026, 7, 6, 10, 17).getTime();
    render(
      <WorkOrderIntentTranscript
        organizationId="org-1"
        messages={[
          { id: "plan-1", kind: "plan", role: "plan", score: 4, createdAtMs: sentAt },
          { id: "agent-1", kind: "text", role: "agent", text: "I updated the plan.", activityId: "activity-1" },
        ]}
        activities={[
          {
            id: "activity-1",
            provider: "codex",
            status: "passed",
            sequence: 1,
            items: [
              {
                type: "tool",
                id: "command-1",
                kind: "bash",
                name: "Bash",
                input: "ls",
                output: "",
                outputStreams: [],
                status: "passed",
                truncated: false,
              },
            ],
            truncated: false,
          },
        ]}
      />,
    );

    expect(screen.queryByRole("time")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-activity-activity-1")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-plan-updated")).not.toBeInTheDocument();
  });

  it("shows the original request time only when that time is valid", () => {
    const sentAt = new Date(2026, 7, 6, 10, 17);
    const { rerender } = renderRequest(sentAt.toISOString());

    const request = screen.getByTestId("split-run-description");
    expectHoverSentTime(within(request).getByRole("time"), sentAt, "overlay");
    expect(request).toHaveTextContent("Show the next action on the empty billing page.");
    request.focus();
    expect(request).toHaveFocus();

    rerender(requestElement("not-a-date"));
    expect(within(screen.getByTestId("split-run-description")).queryByRole("time")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-description")).not.toHaveAttribute("tabindex");
  });
});

function expectHoverSentTime(time: HTMLElement, sentAt: Date, placement: "beside" | "overlay") {
  expect(time).toHaveTextContent(`Sent ${formatWorkOrderDateTime(sentAt)}`);
  expect(time).toHaveAttribute("dateTime", sentAt.toISOString());
  expect(time).toHaveAttribute("title", sentAt.toLocaleString());
  expect(time).toHaveClass(
    "absolute",
    "pointer-events-none",
    "opacity-0",
    "group-hover/message:opacity-100",
    "group-focus/message:opacity-100",
  );
  const row = time.closest("[class*='group/message']");
  expect(row).toHaveClass("group/message", "relative");
  expect(row).toHaveAttribute("tabindex", "0");
  if (placement === "beside") {
    expect(time).toHaveClass("end-[calc(100%+0.5rem)]");
    expect(time).not.toHaveClass("bg-background/95");
    return;
  }
  expect(time).toHaveClass("bg-background/95", "end-2");
}

const REQUEST_VIEW: CreateWithAgentView = {
  repository: "acme/payments",
  machineStatus: "waiting",
  canvasId: "",
  canvasRunId: "",
  executionId: "",
  messages: [],
  composer: "",
  created: [],
  right: { kind: "empty" },
  endConfirmOpen: false,
  selectableModelKey: "",
  refining: false,
};

function requestElement(createdAt?: string) {
  return (
    <TooltipProvider>
      <WorkOrderIntentRequest
        title="Clearer empty state"
        description="Show the next action on the empty billing page."
        createdAt={createdAt}
        analysis={{
          organizationId: "org-1",
          view: REQUEST_VIEW,
          composer: "",
          canSend: false,
          onComposerChange: () => undefined,
          onSend: () => undefined,
          onSubmitSurvey: () => undefined,
        }}
      />
    </TooltipProvider>
  );
}

function renderRequest(createdAt?: string) {
  return render(requestElement(createdAt));
}
