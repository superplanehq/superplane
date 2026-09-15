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

  it("shows survey answers as question and answer in a stronger bubble", () => {
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
    expect(answer).toHaveTextContent("What should the agent build?");
    expect(answer).toHaveTextContent("Custom styled modal");
    expect(answer).toHaveTextContent("How is the work done?");
    expect(answer).toHaveTextContent("Reviewer approves by taste");
    expect(answer).toHaveClass("sp-survey-card");
    expect(answer.className).toContain("border");
    expect(answer.parentElement).toHaveClass("justify-end");
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.youSurvey)).not.toBeInTheDocument();
  });

  it("shows composer notes with the sender given name and avatar", () => {
    renderTranscript([
      { id: "user-1", kind: "text", role: "user", userId: "user-ada", text: "Keep the current dark theme." },
    ]);

    const note = screen.getByTestId("split-run-intent-user-note");
    expect(note).toHaveClass("sp-user-note");
    expect(note).toHaveTextContent("Ada");
    expect(within(note).getByRole("img", { name: "Ada Lovelace" })).toHaveAttribute(
      "src",
      "https://example.com/ada.png",
    );
    expect(note).toHaveTextContent("Keep the current dark theme.");
    expect(note.parentElement).toHaveClass("justify-end");
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.you)).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
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

  it("omits the sender header when the message has no user id", () => {
    renderTranscript([{ id: "user-1", kind: "text", role: "user", text: "Keep the current dark theme." }]);

    const note = screen.getByTestId("split-run-intent-user-note");
    expect(note).toHaveTextContent("Keep the current dark theme.");
    expect(note).not.toHaveTextContent("Ada");
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.you)).not.toBeInTheDocument();
  });

  it("keeps agent lines on the left", () => {
    renderTranscript([{ id: "agent-1", kind: "text", role: "agent", text: "I updated the plan." }]);

    const agent = screen.getByText("I updated the plan.");
    expect(agent.closest(".justify-end")).toBeNull();
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
});
