import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { WorkOrderIntentTranscript } from "./WorkOrderIntentTranscript";

describe("WorkOrderIntentTranscript", () => {
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
});
