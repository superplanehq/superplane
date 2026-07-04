import { createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createSystemMessage } from "@/components/AgentSidebar/systemMessages";
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
          content: "SUPERPLANE_URL=https://app.test superplane apps canvas get canvas-1 --draft",
          toolName: "bash",
          toolCallId: `call-${status}`,
          toolStatus: status,
          createdAt: null,
        },
      ],
    },
  ];
}

function userMessage(content: string): MessageGroup[] {
  return [
    {
      type: "message",
      message: {
        id: "user-message",
        role: "user",
        content,
        toolName: "",
        toolCallId: "",
        toolStatus: "",
        createdAt: null,
      },
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

describe("ConversationTranscript user messages", () => {
  it("renders attached images as linked thumbnails", () => {
    const groups: MessageGroup[] = [
      {
        type: "message",
        message: {
          id: "user-with-image",
          role: "user",
          content: "fix this",
          toolName: "",
          toolCallId: "",
          toolStatus: "",
          images: [{ mediaType: "image/png", url: "/api/v1/agents/chats/c-1/messages/user-with-image/images/0" }],
          createdAt: null,
        },
      },
    ];

    render(<ConversationTranscript {...baseProps} messageGroups={groups} />);

    const image = screen.getByRole("img", { name: "attachment" });
    expect(image).toHaveAttribute("src", "/api/v1/agents/chats/c-1/messages/user-with-image/images/0");
  });

  it("keeps compact user messages sticky", () => {
    render(<ConversationTranscript {...baseProps} messageGroups={userMessage("Build a release workflow")} />);

    expect(screen.getByTestId("agent-user-message").parentElement).toHaveClass("sticky");
  });

  it("does not keep user messages with image attachments sticky", () => {
    const groups: MessageGroup[] = [
      {
        type: "message",
        message: {
          id: "user-with-image",
          role: "user",
          content: "",
          toolName: "",
          toolCallId: "",
          toolStatus: "",
          images: [{ mediaType: "image/png", url: "/img/0" }],
          createdAt: null,
        },
      },
    ];

    render(<ConversationTranscript {...baseProps} messageGroups={groups} />);

    expect(screen.getByTestId("agent-user-message").parentElement).not.toHaveClass("sticky");
  });

  it("does not keep long user messages sticky", () => {
    const longPrompt = [
      "My idea is: Create an elixir clusterautoprovisioning app with enough detail to wrap across multiple lines.",
      "",
      "- CI/CD deployment on push to main.",
      "- Metric integration that provisions another node when usage gets high.",
      "- Backup postgres too.",
      "",
      "Also suggest other workflows in the canvas if you find interesting ones for the hackaton.",
    ].join("\n");

    render(<ConversationTranscript {...baseProps} messageGroups={userMessage(longPrompt)} />);

    expect(screen.getByTestId("agent-user-message").parentElement).not.toHaveClass("sticky");
  });
});

describe("ConversationTranscript hidden messages", () => {
  it("does not render an empty turn above the thinking row", () => {
    render(
      <ConversationTranscript
        {...baseProps}
        messageGroups={[
          {
            type: "message",
            message: {
              id: "system-notification",
              role: "user",
              content: createSystemMessage("Canvas changed"),
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              createdAt: null,
            },
          },
        ]}
        showThinking
      />,
    );

    const transcriptBody = screen.getByTestId("agent-thinking").parentElement;
    expect(transcriptBody?.firstElementChild).toBe(screen.getByTestId("agent-thinking"));
  });

  it("does not render an empty turn above command groups", () => {
    render(
      <ConversationTranscript
        {...baseProps}
        messageGroups={[
          {
            type: "message",
            message: {
              id: "system-message",
              role: "system",
              content: "Internal event",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              createdAt: null,
            },
          },
          ...toolGroup("started"),
        ]}
      />,
    );

    const transcriptBody = screen.getByTestId("agent-tool-group").parentElement;
    expect(transcriptBody?.firstElementChild).toBe(screen.getByTestId("agent-tool-group"));
  });
});
