/** Deterministic agent chat seed for AppPage Storybook stories. */

export const STORYBOOK_AGENT_CHAT_ID = "storybook-agent-chat";

/** Fired after a simulated agent send so Storybook can refetch the transcript. */
export const STORYBOOK_AGENT_MESSAGES_UPDATED_EVENT = "storybook:agent-messages-updated";

type StorybookAgentMessage = {
  id: string;
  role: string;
  content: string;
  createdAt: string;
};

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
    messages: seedMessages(),
    hasMore: false,
  };
}

function seedMessages(): StorybookAgentMessage[] {
  return [
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
  ];
}

/**
 * In-memory agent transcript for a Storybook fixture session. POST appends a
 * user message plus a short simulated assistant reply so suggestion sends are
 * visible without a real agent backend.
 */
export function createStorybookAgentMessageStore(initialMessages?: Array<Record<string, unknown>>) {
  const messages: StorybookAgentMessage[] = (initialMessages?.length ? initialMessages : seedMessages()).map(
    (message) => ({
      id: String(message.id ?? ""),
      role: String(message.role ?? "assistant"),
      content: String(message.content ?? ""),
      createdAt: String(message.createdAt ?? new Date().toISOString()),
    }),
  );
  let seq = 0;

  return {
    list() {
      return { messages: [...messages], hasMore: false };
    },
    send(body: unknown) {
      const content =
        body && typeof body === "object" && "content" in body
          ? String((body as { content?: unknown }).content ?? "")
          : "";
      seq += 1;
      const createdAt = new Date().toISOString();
      const userMessage: StorybookAgentMessage = {
        id: `storybook-sent-${seq}`,
        role: "user",
        content,
        createdAt,
      };
      const assistantMessage: StorybookAgentMessage = {
        id: `storybook-reply-${seq}`,
        role: "assistant",
        content: simulateAssistantReply(content),
        createdAt: new Date(Date.now() + 1).toISOString(),
      };
      messages.push(userMessage, assistantMessage);
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent(STORYBOOK_AGENT_MESSAGES_UPDATED_EVENT));
      }
      return { message: userMessage };
    },
    reset() {
      messages.splice(0, messages.length, ...seedMessages());
      seq = 0;
    },
  };
}

function simulateAssistantReply(userContent: string): string {
  const preview = userContent.trim().replace(/\s+/g, " ").slice(0, 120);
  if (!preview) {
    return "Got it — I'll take a look. _(Storybook simulation)_";
  }
  return `Got it — I'll work on: “${preview}${userContent.trim().length > 120 ? "…" : ""}”. _(Storybook simulation)_`;
}

type StorybookAgentMessageStore = ReturnType<typeof createStorybookAgentMessageStore>;

/** Handle stateful agent chat routes for Storybook fixture fetches. */
export function matchStorybookAgentMessageRoute(args: {
  pathname: string;
  method: string;
  body: unknown;
  agentMessages: StorybookAgentMessageStore;
  resetChat: () => unknown;
}): { json: unknown } | null {
  const { pathname, method, body, agentMessages, resetChat } = args;
  if (/^\/api\/v1\/agents\/chats\/[^/]+\/messages$/.test(pathname)) {
    if (method === "POST") return { json: agentMessages.send(body) };
    if (method === "GET") return { json: agentMessages.list() };
  }
  if (/^\/api\/v1\/agents\/canvases\/[^/]+\/chat\/reset$/.test(pathname) && method === "POST") {
    agentMessages.reset();
    return { json: resetChat() };
  }
  return null;
}
