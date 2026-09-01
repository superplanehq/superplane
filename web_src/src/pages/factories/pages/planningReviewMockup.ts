export interface PlanningReviewConcurrency {
  max: string;
  key: string;
}

export type PlanningReviewStepKind = "bash" | "prompt";

export interface PlanningReviewStep {
  name: string;
  type: PlanningReviewStepKind;
  command?: string;
  prompt?: string;
  workingDirectory?: string;
}

export interface PlanningReviewComponent {
  id: string;
  title: string;
  description: string;
  expanded: boolean;
  configuration: Record<string, unknown>;
  concurrency: PlanningReviewConcurrency;
}

export interface PlanningReviewDraft {
  title: string;
  components: PlanningReviewComponent[];
}

const implementationSteps: PlanningReviewStep[] = [
  { name: "Clone Repo", type: "bash", command: 'git clone --depth 1 "$REPO_URL" repo' },
  {
    name: "Implementation",
    type: "prompt",
    prompt: "Create a working branch and implement the approved plan.",
    workingDirectory: "repo",
  },
  { name: "Commit and Push", type: "bash", command: "git push origin HEAD", workingDirectory: "repo" },
];

const implementationConfiguration: Record<string, unknown> = {
  machineType: "e1-large-amd64",
  credentials: {
    source: "integration",
    integration: "claude-superplane-apps",
  },
  model: "sonnet",
  steps: implementationSteps,
  workingDirectory: "/tmp/repo",
  environmentFrom: [{ source: "integration", integration: "github-superplanehq" }],
  environment: [
    { name: "REPO_URL", valueSource: "literal", value: "{{ order().repository_url }}" },
    { name: "BASE", valueSource: "literal", value: "{{ order().default_branch }}" },
  ],
  executionTimeoutSeconds: 3600,
};

export const PLANNING_REVIEW_DRAFT: PlanningReviewDraft = {
  title: "Planning review",
  components: [
    {
      id: "implementation-agent",
      title: "Agent - Implement from order description",
      description: "Implement the approved plan and prepare the branch for review.",
      expanded: true,
      configuration: implementationConfiguration,
      concurrency: { max: "5", key: "ci-{{ $.data.branch }}" },
    },
  ],
};

/** One editing window shows one agent. Extra agents stay on the canvas. */
export function singleAgentDraft(draft: PlanningReviewDraft): PlanningReviewDraft {
  const component = draft.components.find((entry) => entry.expanded) ?? draft.components[0];
  if (!component) {
    return { ...draft, components: [] };
  }
  return { ...draft, components: [{ ...component, expanded: true }] };
}
