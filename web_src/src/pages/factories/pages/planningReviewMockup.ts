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
  { name: "Set Up Git User", type: "bash", command: 'git config --global user.email "agent@superplane.com"' },
  {
    name: "Provide order",
    type: "bash",
    command: "echo $ORDER_DESCRIPTION | base64 -d > /tmp/ORDER.md\ncat /tmp/ORDER.md",
    workingDirectory: "repo",
  },
  { name: "Provide Plan", type: "bash", command: "echo $PLAN | base64 -d > /tmp/PLAN.md" },
  { name: "Checkout Branch", type: "bash", command: "git checkout $BRANCH", workingDirectory: "repo" },
  { name: "Set Up DCO Signing", type: "bash", command: "git config commit.gpgsign false", workingDirectory: "repo" },
  {
    name: "Implementation",
    type: "prompt",
    prompt: "Implement the approved plan. Make the smallest complete change and update the tests.",
    workingDirectory: "repo",
  },
  { name: "Commit and Push", type: "bash", command: "git push origin HEAD", workingDirectory: "repo" },
];

const planSteps: PlanningReviewStep[] = [
  { name: "Clone Repo", type: "bash", command: "git clone $REPO repo" },
  { name: "Write Implementation Plan", type: "prompt", prompt: "Write a concise implementation plan." },
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
    { name: "REPO", valueSource: "literal", value: "superplanehq/superplane" },
    { name: "BRANCH", valueSource: "literal", value: '{{ $["Create Branch"].data.result.branch }}' },
    { name: "ORDER_DESCRIPTION", valueSource: "literal", value: "{{ toBase64(order().description) }}" },
    {
      name: "PLAN",
      valueSource: "literal",
      value: '{{ toBase64(find(order().artifacts, {#.type == "markdown" && #.data.title == "PLAN.md"}).data.body) }}',
    },
  ],
  executionTimeoutSeconds: 3600,
};

export const PLANNING_REVIEW_DRAFT: PlanningReviewDraft = {
  title: "Planning review",
  components: [
    {
      id: "plan-agent",
      title: "Agent - Plan for GH Issue",
      description: "Read the issue and write an implementation plan.",
      expanded: false,
      configuration: { ...implementationConfiguration, steps: planSteps },
      concurrency: { max: "1", key: "" },
    },
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
