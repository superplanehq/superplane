import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMessage } from "../createWithAgentTypes";
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

function expectAvatarBeforeAnswer(pick: HTMLElement, avatarName: string, answer: string) {
  const avatar = within(pick).getByRole("img", { name: avatarName });
  const answerText = within(pick).getByText(answer);
  expect(avatar.compareDocumentPosition(answerText) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
}

describe("WorkOrderIntentTranscript", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([
    { role: "agent" as const, origin: undefined, testId: "split-run-intent-agent-message" },
    { role: "user" as const, origin: undefined, testId: "split-run-intent-user-note" },
    { role: "user" as const, origin: "survey" as const, testId: "split-run-intent-survey-answer" },
  ])("shows only the time while hovering $role $origin messages", async ({ role, origin, testId }) => {
    const user = userEvent.setup();
    const createdAtMs = new Date(2026, 9, 13, 14, 25, 36).getTime();
    renderTranscript([{ id: "message", kind: "text", role, origin, text: "Keep the current scope.", createdAtMs }]);

    const time = screen.getByText("14:25");
    expect(time).toHaveAttribute("dateTime", new Date(createdAtMs).toISOString());
    expect(time).not.toBeVisible();
    await user.hover(screen.getByTestId(testId));
    expect(time).toBeVisible();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
    await user.unhover(screen.getByTestId(testId));
    expect(time).not.toBeVisible();
  });

  it.each([undefined, Number.NaN, Number.POSITIVE_INFINITY])(
    "omits an unavailable hover timestamp %s",
    (createdAtMs) => {
      renderTranscript([{ id: "message", kind: "text", role: "agent", text: "Still readable.", createdAtMs }]);
      expect(screen.getByText("Still readable.")).toBeInTheDocument();
      expect(document.querySelector("time")).toBeNull();
    },
  );

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
    expect(within(answer).queryByText("Answered by")).not.toBeInTheDocument();
    expect(within(answer).queryByText("Ada")).not.toBeInTheDocument();
    expect(answer).toHaveTextContent("What should the agent build?");
    expect(answer).toHaveTextContent("Custom styled modal");
    expect(answer).toHaveTextContent("How is the work done?");
    expect(answer).toHaveTextContent("Reviewer approves by taste");
    const picks = within(answer).getAllByTestId("split-run-intent-survey-pick");
    expect(within(picks[0]).queryByText("What should the agent build?")).not.toBeInTheDocument();
    expect(within(picks[1]).queryByText("How is the work done?")).not.toBeInTheDocument();
    expectAvatarBeforeAnswer(picks[0], "Ada Lovelace", "Custom styled modal");
    expectAvatarBeforeAnswer(picks[1], "Ada Lovelace", "Reviewer approves by taste");
    expect(
      within(answer).getByText("What should the agent build?").compareDocumentPosition(picks[0]) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    for (const pick of picks) {
      expect(pick).toHaveClass("sp-chat-outgoing");
      expect(pick.querySelector("[title='Ada Lovelace']")).not.toBeNull();
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
    const survey = screen.getByTestId("split-run-intent-survey-answer");
    expect(within(survey).queryByText("Ada")).not.toBeInTheDocument();
    expect(within(survey).queryByText("Answered by")).not.toBeInTheDocument();
    expectAvatarBeforeAnswer(within(survey).getByTestId("split-run-intent-survey-pick"), "Ada Lovelace", "One file");
    expect(notes[1]).toHaveTextContent("Ada");
  });

  it("shows a survey pick without an avatar when the sender cannot be resolved", () => {
    renderTranscript([
      {
        id: "user-1",
        kind: "text",
        role: "user",
        origin: "survey",
        userId: "user-missing",
        text: "What should the agent build? Custom styled modal",
      },
    ]);

    const pick = screen.getByTestId("split-run-intent-survey-pick");
    expect(pick).toHaveTextContent("Custom styled modal");
    expect(within(pick).queryByRole("img")).not.toBeInTheDocument();
    expect(screen.queryByText("Answered by")).not.toBeInTheDocument();
  });

  it("shows the sender avatar in a survey answer that is plain text", () => {
    renderTranscript([
      {
        id: "user-1",
        kind: "text",
        role: "user",
        origin: "survey",
        userId: "user-ada",
        text: "Add separate lists per animal.",
      },
    ]);

    expectAvatarBeforeAnswer(
      screen.getByTestId("split-run-intent-survey-pick"),
      "Ada Lovelace",
      "Add separate lists per animal.",
    );
  });

  it("does not show an avatar on a skipped survey answer", () => {
    renderTranscript([
      {
        id: "user-1",
        kind: "text",
        role: "user",
        origin: "survey",
        userId: "user-ada",
        text: "What should the agent build? Custom styled modal\nHow is the work done? skipped",
      },
    ]);

    const answer = screen.getByTestId("split-run-intent-survey-answer");
    const questions = within(answer).getAllByRole("listitem");
    const skippedQuestion = questions.find((question) =>
      within(question).queryByText(CREATE_WITH_AGENT_COPY.surveyAnswerSkipped),
    );
    const answeredQuestion = questions.find((question) => within(question).queryByText("Custom styled modal"));
    if (!skippedQuestion || !answeredQuestion) {
      throw new Error("expected a skipped question and an answered question");
    }
    expect(within(skippedQuestion).queryByRole("img")).not.toBeInTheDocument();
    expectAvatarBeforeAnswer(
      within(answeredQuestion).getByTestId("split-run-intent-survey-pick"),
      "Ada Lovelace",
      "Custom styled modal",
    );
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
});
