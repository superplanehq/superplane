import type { Meta, StoryObj } from "@storybook/react-vite";

import { CheckKindLabel, CheckOutcomeChip } from "./CheckOutcomeChip";
import { EnforcementBadge, SeverityBadge } from "./SeverityBadge";
import { VerificationStatusBadge } from "./VerificationStatusBadge";
import type { CheckOutcome, FindingSeverity, VerificationRunStatus } from "./types";

const RUN_STATUSES: VerificationRunStatus[] = ["running", "passed", "failed"];
const CHECK_OUTCOMES: CheckOutcome[] = ["running", "passed", "failed", "skipped"];
const SEVERITIES: FindingSeverity[] = ["high", "medium", "low"];

/**
 * Status primitives for the verification designs: run status badges, check
 * outcome chips, check kind labels, severity badges, and enforcement badges.
 */
const meta = {
  title: "Factories/Verification/StatusPrimitives",
  component: VerificationStatusBadge,
  parameters: { layout: "centered" },
} satisfies Meta<typeof VerificationStatusBadge>;

export default meta;

type Story = StoryObj<typeof meta>;

/** All verification run statuses. */
export const RunStatuses: Story = {
  args: { status: "running" },
  render: () => (
    <div className="flex items-center gap-2">
      {RUN_STATUSES.map((status) => (
        <VerificationStatusBadge key={status} status={status} />
      ))}
    </div>
  ),
};

/** All check outcomes plus the agent/command kind labels. */
export const CheckOutcomes: Story = {
  args: { status: "running" },
  render: () => (
    <div className="flex flex-col items-start gap-3">
      <div className="flex items-center gap-2">
        {CHECK_OUTCOMES.map((outcome) => (
          <CheckOutcomeChip key={outcome} outcome={outcome} />
        ))}
      </div>
      <div className="flex items-center gap-4">
        <CheckKindLabel kind="agent" />
        <CheckKindLabel kind="command" />
      </div>
    </div>
  ),
};

/** All finding severities plus the enforcement badges. */
export const Severities: Story = {
  args: { status: "running" },
  render: () => (
    <div className="flex flex-col items-start gap-3">
      <div className="flex items-center gap-2">
        {SEVERITIES.map((severity) => (
          <SeverityBadge key={severity} severity={severity} />
        ))}
      </div>
      <div className="flex items-center gap-2">
        <EnforcementBadge enforcement="blocking" />
        <EnforcementBadge enforcement="advisory" />
      </div>
    </div>
  ),
};
