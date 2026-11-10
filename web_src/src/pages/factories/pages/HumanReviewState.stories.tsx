import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { HumanReviewState } from "./HumanReviewState";

const meta = {
  title: "Factories/Components/HumanReviewState",
  component: HumanReviewState,
  args: {
    pullRequest: {
      id: "pr",
      humanReview: { status: "waiting_assessment", blockingReason: "Waiting for merge confidence." },
    },
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="w-[320px] max-w-full rounded-lg border border-border bg-background p-3">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof HumanReviewState>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Waiting: Story = {};
export const ApprovalRequired: Story = {
  args: {
    pullRequest: {
      humanReview: {
        status: "decided",
        reviewRequired: true,
        rationale: "A database migration needs review from its owner.",
        selectedReviewers: [{ id: "7", login: "alex", reason: "Owns the affected database files." }],
        blockingReason: "A selected reviewer must approve the current head commit.",
        checkUrl: "https://github.com/acme/app/runs/7",
      },
    },
  },
};
export const DetailedAssessment: Story = {
  args: {
    pullRequest: {
      humanReview: {
        status: "decided",
        reviewRequired: true,
        rationale:
          "The drift check found changes outside the task scope. The pull request modifies application code, but the task only requests documentation. The code owners are already requested and eligible. No additional reviewers are needed.",
        selectedReviewers: [
          { id: "7", login: "shiroyasha", reason: "Configured default reviewer with repository write access." },
          { id: "8", login: "alex", reason: "Owns the affected application files." },
        ],
        blockingReason: "A selected reviewer must approve the current head commit.",
        checkUrl: "https://github.com/acme/app/runs/7",
      },
    },
  },
};
export const Approved: Story = {
  args: {
    pullRequest: {
      humanReview: {
        status: "decided",
        reviewRequired: true,
        approvalIds: ["8"],
        rationale: "The selected reviewer approved this commit.",
      },
    },
  },
};
export const NoReview: Story = {
  args: {
    pullRequest: {
      humanReview: {
        status: "decided",
        reviewRequired: false,
        rationale: "The complete assessment found no concerns.",
      },
    },
  },
};
export const Failed: Story = {
  args: {
    pullRequest: {
      humanReview: {
        status: "failed",
        blockingReason: "Merge confidence did not produce a complete assessment. Run merge confidence again.",
        publicationError: "GitHub is unavailable",
      },
    },
  },
};
export const Disabled: Story = {
  args: { pullRequest: { humanReview: { status: "disabled", rationale: "Human review enforcement is disabled." } } },
};
