/** Deterministic agent chat seed for AppPage Storybook stories. */

export const STORYBOOK_AGENT_CHAT_ID = "storybook-agent-chat";

export function buildStorybookAgentChat(canvasId: string) {
  return {
    chat: {
      id: STORYBOOK_AGENT_CHAT_ID,
      canvasId,
      provider: "claude",
      status: "idle",
      createdAt: "2026-07-15T12:00:00.000Z",
      updatedAt: "2026-07-15T12:05:00.000Z",
    },
  };
}

export function buildStorybookAgentMessages() {
  return {
    messages: [
      {
        id: "storybook-msg-user-1",
        role: "user",
        content: "Can you summarize what this Software Factory canvas does?",
        createdAt: "2026-07-15T12:01:00.000Z",
      },
      {
        id: "storybook-msg-assistant-1",
        role: "assistant",
        content:
          "This canvas watches GitHub issues labeled for implementation, opens a branch and PR, then runs the implementation and CI loop until the work is ready to review.",
        createdAt: "2026-07-15T12:01:20.000Z",
      },
      {
        id: "storybook-msg-user-2",
        role: "user",
        content: "Which node starts the implementation runner?",
        createdAt: "2026-07-15T12:02:00.000Z",
      },
      {
        id: "storybook-msg-assistant-2",
        role: "assistant",
        content:
          "The **Implementation** node (`runner-implement`) is the main runner. Upstream steps create the branch/PR and notify Discord before it starts.",
        createdAt: "2026-07-15T12:02:25.000Z",
      },
    ],
    hasMore: false,
  };
}
