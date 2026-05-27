import { createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ConversationTranscript } from "./AgentConversationTranscript";
import type { MessageGroup } from "./agentMessageGroups";

const baseProps = {
  error: null,
  canvasId: "canvas-1",
  organizationId: "org-1",
  isLoading: false,
  isLoadingMore: false,
  onAction: vi.fn(async () => undefined),
  onStartBuilding: vi.fn(async () => undefined),
  scrollRef: createRef<HTMLDivElement>(),
  showThinking: false,
};

function toolGroup(status: string): MessageGroup[] {
  return [
    {
      type: "tool-group",
      messages: [
        {
          id: `tool-${status}`,
          role: "tool",
          content: "SUPERPLANE_URL=https://app.test superplane canvases get canvas-1 --draft",
          toolName: "bash",
          toolCallId: `call-${status}`,
          toolStatus: status,
          createdAt: null,
        },
      ],
    },
  ];
}

describe("ConversationTranscript command groups", () => {
  it("collapses completed command groups by default", () => {
    render(<ConversationTranscript {...baseProps} messageGroups={toolGroup("finished")} />);

    expect(screen.getByText("Ran 1 command")).toBeInTheDocument();
    expect(screen.queryByText(/SUPERPLANE_URL/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByText("Ran 1 command"));
    expect(screen.getByText(/SUPERPLANE_URL/)).toBeInTheDocument();
  });

  it("keeps running command groups expanded", () => {
    render(<ConversationTranscript {...baseProps} messageGroups={toolGroup("started")} />);

    expect(screen.getByText("Running command...")).toBeInTheDocument();
    expect(screen.getByText(/SUPERPLANE_URL/)).toBeInTheDocument();
  });
});
