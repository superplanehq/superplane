import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { WorkOrderIntentTranscript } from "./WorkOrderIntentTranscript";

let notifyResize: () => void;

class MockResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    notifyResize = () => callback([], this as unknown as ResizeObserver);
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}

describe("WorkOrderIntentTranscript", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not restream the last agent line after a user reply", () => {
    render(
      <WorkOrderIntentTranscript
        streaming
        messages={[
          { id: "agent-1", kind: "text", role: "agent", text: "I asked one survey question." },
          { id: "user-1", kind: "text", role: "user", origin: "survey", text: "Add separate lists per animal." },
        ]}
      />,
    );

    const transcript = screen.getByTestId("split-run-intent-transcript");
    expect(transcript.querySelector(".sp-stream-text")).toBeNull();
    expect(transcript.querySelector(".sp-text-reveal")).not.toBeNull();
  });

  it("streams the latest agent line while that turn is still open", () => {
    render(
      <WorkOrderIntentTranscript
        streaming
        messages={[{ id: "agent-1", kind: "text", role: "agent", text: "I asked one survey question." }]}
      />,
    );

    expect(screen.getByTestId("split-run-intent-transcript").querySelector(".sp-stream-text")).not.toBeNull();
  });

  it("shows survey answers as question and answer in a stronger bubble", () => {
    render(
      <WorkOrderIntentTranscript
        messages={[
          {
            id: "user-1",
            kind: "text",
            role: "user",
            origin: "survey",
            text: "What should the agent build? Custom styled modal\nHow is the work done? Reviewer approves by taste",
          },
        ]}
      />,
    );

    const answer = screen.getByTestId("split-run-intent-survey-answer");
    expect(answer).toHaveTextContent(CREATE_WITH_AGENT_COPY.youSurvey);
    expect(answer).toHaveTextContent("What should the agent build?");
    expect(answer).toHaveTextContent("Custom styled modal");
    expect(answer).toHaveTextContent("How is the work done?");
    expect(answer).toHaveTextContent("Reviewer approves by taste");
    expect(answer).toHaveClass("sp-survey-card");
    expect(answer.className).toContain("border");
  });

  it("shows composer notes in the same accent bubble as the survey", () => {
    render(
      <WorkOrderIntentTranscript
        messages={[{ id: "user-1", kind: "text", role: "user", text: "Keep the current dark theme." }]}
      />,
    );

    const note = screen.getByTestId("split-run-intent-user-note");
    expect(note).toHaveClass("sp-user-note");
    expect(note).toHaveTextContent(CREATE_WITH_AGENT_COPY.you);
    expect(note).toHaveTextContent("Keep the current dark theme.");
    expect(within(note).queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
  });

  it("collapses a long composer note and expands it on Show more", async () => {
    const user = userEvent.setup();
    render(
      <WorkOrderIntentTranscript
        messages={[
          {
            id: "user-1",
            kind: "text",
            role: "user",
            text: "Need a payment method.\n".repeat(40),
          },
        ]}
      />,
    );

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
});
