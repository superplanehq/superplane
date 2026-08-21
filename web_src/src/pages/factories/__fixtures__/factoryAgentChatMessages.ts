export function factoryAgentChatMessages() {
  return {
    messages: [
      {
        id: "factory-agent-msg-user-1",
        role: "user",
        content: "Can you change this Refund Implementer canvas so Semaphore CI retries after a failure?",
        createdAt: "2026-07-15T12:01:00.000Z",
      },
      {
        id: "factory-agent-msg-assistant-1",
        role: "assistant",
        content:
          "I can help you change this Refund Implementer canvas. Tell me which step to change. I will describe the YAML update. _(Storybook simulation)_",
        createdAt: "2026-07-15T12:01:20.000Z",
      },
    ],
    hasMore: false,
  };
}
