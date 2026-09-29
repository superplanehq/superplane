export const IMPLEMENTER_AGENT_STEPS = [
  { name: "Clone Repo", type: "bash" as const, command: "git clone $REPO repo" },
  { name: "Implement", type: "prompt" as const, prompt: "Implement the plan.", workingDirectory: "repo" },
];

export const implementerCanvas = {
  metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} },
      {
        id: "implementation-agent",
        name: "Implement From Task Description",
        type: "TYPE_ACTION",
        component: "runnerClaudeCode",
        concurrency: { max: 5 },
        configuration: {
          machineType: "e1-large-amd64",
          model: "sonnet",
          credentials: { source: "integration", integration: { name: "claude" } },
          steps: IMPLEMENTER_AGENT_STEPS,
        },
      },
    ],
    edges: [],
  },
};

export const canvasWithoutAgent = {
  metadata: { id: "app-no-agent", liveVersionId: "version-live" },
  spec: {
    nodes: [{ id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} }],
    edges: [],
  },
};

export function canvasQuery(data: typeof implementerCanvas | typeof canvasWithoutAgent) {
  return { data, isPending: false, isError: false };
}
