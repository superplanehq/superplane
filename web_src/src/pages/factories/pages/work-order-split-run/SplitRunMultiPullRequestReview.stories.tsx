import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState, type ReactNode } from "react";

import type { FactoriesFactoryPullRequest } from "@/api-client";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { pullRequestChecksKey } from "../../lib/mergeConfidenceScore";
import type { WorkOrderCheckLevel, WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { PullRequestChecksContext } from "./redesign/pullRequestChecksContext";
import { MultiPullRequestReviewNote } from "./SplitRunMultiPullRequestReview";

const meta = {
  title: "Factories/Pages/Task Split Run/Multi-repo pull request review",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story, context) => (
      <StoryQueryClient key={context.id}>
        <ComponentStoryShell className="min-h-svh bg-muted/30 p-6">
          <Story />
        </ComponentStoryShell>
      </StoryQueryClient>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

/** Each story gets its own cache, so the merge story does not leak merge controls into the others. */
function StoryQueryClient({ children }: { children: ReactNode }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

const API_REPOSITORY = "forestileao/arq-comp-microprocessor";
const CHAIN_REPOSITORY = "forestileao/actor-based-blockchain";
const DOCS_REPOSITORY = "forestileao/docs";

function pullRequest(id: string, repository: string, number: string, title?: string): FactoriesFactoryPullRequest {
  return {
    id,
    provider: "PROVIDER_GITHUB",
    state: "STATE_OPEN",
    repository,
    number,
    title,
    url: `https://github.com/${repository}/pull/${number}`,
  };
}

const API_PULL_REQUEST = pullRequest(
  "pr-api",
  API_REPOSITORY,
  "4",
  "docs: add root docs.md with quick start, testing, and layout",
);
const CHAIN_PULL_REQUEST = pullRequest(
  "pr-chain",
  CHAIN_REPOSITORY,
  "1",
  "docs: add root docs.md with overview, quick start, testing, and layout",
);
const DOCS_PULL_REQUEST = pullRequest("pr-docs", DOCS_REPOSITORY, "128", "docs: link the new repository guides");

const CHECK_NAMES = ["Risk score", "Code quality", "Security", "Test coverage", "Reversibility"];

function checks(repository: string, levels: WorkOrderCheckLevel[]): WorkOrderCheckPresentation[] {
  return levels.map((level, index) => {
    const assessed = level !== "neutral";
    return {
      id: `${repository}-${index}`,
      key: CHECK_NAMES[index]?.toLowerCase().replaceAll(" ", "-"),
      repository,
      name: CHECK_NAMES[index] ?? `Check ${index + 1}`,
      score: { positive: 5, neutral: 0, caution: 3, critical: 1 }[level],
      scoreSupplied: assessed,
      maxScore: 5,
      level,
      summary: assessed ? `${CHECK_NAMES[index]} for ${repository}.` : "The check did not report yet.",
      sourceName: "Merge confidence",
    };
  });
}

const PASSED: WorkOrderCheckLevel[] = ["positive", "positive", "positive", "positive", "positive"];
const PARTIAL: WorkOrderCheckLevel[] = ["positive", "positive", "neutral", "neutral", "neutral"];
const NEEDS_REVIEW: WorkOrderCheckLevel[] = ["positive", "caution", "positive", "positive", "positive"];
const NEEDS_ATTENTION: WorkOrderCheckLevel[] = ["critical", "caution", "positive", "positive", "positive"];

/** Merge checks for each repository. A repository without an entry has no merge confidence yet. */
function checksByRepository(entries: [string, WorkOrderCheckLevel[]][]) {
  return new Map(entries.map(([repository, levels]) => [pullRequestChecksKey(repository), checks(repository, levels)]));
}

/** Matches the width of the task summary panel. */
function SummaryPanel({
  byRepository,
  children,
}: {
  byRepository: ReturnType<typeof checksByRepository> | null;
  children: ReactNode;
}) {
  return (
    <PullRequestChecksContext.Provider value={byRepository}>
      <div className="w-[340px] rounded-lg border border-border bg-background p-3">{children}</div>
    </PullRequestChecksContext.Provider>
  );
}

function ReviewNote({
  pullRequests = [API_PULL_REQUEST, CHAIN_PULL_REQUEST],
  byRepository,
  continueFixes,
}: {
  pullRequests?: FactoriesFactoryPullRequest[];
  byRepository: ReturnType<typeof checksByRepository> | null;
  continueFixes?: () => void;
}) {
  return (
    <SummaryPanel byRepository={byRepository}>
      <MultiPullRequestReviewNote
        pullRequests={pullRequests}
        organizationId="org-1"
        factoryId="factory-1"
        orderId="wo-1"
        canAct
        stacked={false}
        continueFixes={continueFixes}
        actionBusy={false}
      />
    </SummaryPanel>
  );
}

export const NoMergeConfidence: Story = {
  name: "Ready — no merge confidence yet",
  render: () => <ReviewNote byRepository={null} />,
};

export const MergeConfidenceRunning: Story = {
  name: "Ready — merge confidence partly reported",
  render: () => (
    <ReviewNote
      byRepository={checksByRepository([
        [API_REPOSITORY, PARTIAL],
        [CHAIN_REPOSITORY, PARTIAL],
      ])}
    />
  ),
};

export const OneRepositoryScored: Story = {
  name: "Ready — only one repository scored",
  render: () => <ReviewNote byRepository={checksByRepository([[API_REPOSITORY, PASSED]])} />,
};

export const AllPassed: Story = {
  name: "Ready — all checks passed",
  render: () => (
    <ReviewNote
      byRepository={checksByRepository([
        [API_REPOSITORY, PASSED],
        [CHAIN_REPOSITORY, PASSED],
      ])}
    />
  ),
};

export const OneNeedsReview: Story = {
  name: "Ready — one repository needs review",
  render: () => (
    <ReviewNote
      byRepository={checksByRepository([
        [API_REPOSITORY, PASSED],
        [CHAIN_REPOSITORY, NEEDS_REVIEW],
      ])}
    />
  ),
};

export const NeedsAttention: Story = {
  name: "Ready — checks need attention",
  render: () => (
    <ReviewNote
      byRepository={checksByRepository([
        [API_REPOSITORY, NEEDS_ATTENTION],
        [CHAIN_REPOSITORY, NEEDS_REVIEW],
      ])}
    />
  ),
};

export const ThreeRepositories: Story = {
  name: "Three repositories — mixed states",
  render: () => (
    <ReviewNote
      pullRequests={[API_PULL_REQUEST, CHAIN_PULL_REQUEST, DOCS_PULL_REQUEST]}
      byRepository={checksByRepository([
        [API_REPOSITORY, PASSED],
        [CHAIN_REPOSITORY, PARTIAL],
      ])}
    />
  ),
};

export const UntitledPullRequests: Story = {
  name: "Pull requests without titles",
  render: () => (
    <ReviewNote
      pullRequests={[pullRequest("pr-api", API_REPOSITORY, "4"), pullRequest("pr-chain", CHAIN_REPOSITORY, "1")]}
      byRepository={checksByRepository([
        [API_REPOSITORY, PASSED],
        [CHAIN_REPOSITORY, PASSED],
      ])}
    />
  ),
};

export const FixesPaused: Story = {
  name: "Automatic fixes paused",
  render: () => (
    <ReviewNote
      byRepository={checksByRepository([
        [API_REPOSITORY, NEEDS_REVIEW],
        [CHAIN_REPOSITORY, PASSED],
      ])}
      continueFixes={() => console.log("continue fixes")}
    />
  ),
};

export const MergeActions: Story = {
  name: "Merge enabled — one ready, one blocked",
  parameters: {
    msw: {
      handlers: [
        http.get("*/account/experimental-features", () =>
          HttpResponse.json({ features: [{ id: "factory_pull_request_merge", released: true }] }),
        ),
        http.get("*/api/v1/organizations/org-1", () =>
          HttpResponse.json({ organization: { metadata: { id: "org-1" } } }),
        ),
        http.get("*/api/v1/factories/factory-1/prs/pr-api/mergeability", () =>
          HttpResponse.json({
            mergeability: { canMerge: true, allowedMethods: ["MERGE_METHOD_SQUASH"], headSha: "abc123" },
          }),
        ),
        http.get("*/api/v1/factories/factory-1/prs/pr-chain/mergeability", () =>
          HttpResponse.json({
            mergeability: {
              canMerge: false,
              blockedReason: "BLOCKED_REASON_CHECKS_UNFINISHED",
              message: "Checks are still running.",
              allowedMethods: ["MERGE_METHOD_SQUASH"],
              headSha: "def456",
            },
          }),
        ),
      ],
    },
  },
  render: () => (
    <ReviewNote
      byRepository={checksByRepository([
        [API_REPOSITORY, PASSED],
        [CHAIN_REPOSITORY, PARTIAL],
      ])}
    />
  ),
};
