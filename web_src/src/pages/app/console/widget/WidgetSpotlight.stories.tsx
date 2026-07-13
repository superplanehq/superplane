import type { ComponentProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TypedPanelShell } from "../TypedPanelShell";
import { PanelFrame } from "../__stories__/storyDecorators";

import { WidgetSpotlight } from "./WidgetSpotlight";

/**
 * Spotlight panel renderer — a single record blown up into a hero banner.
 * Slots are generic (who, what + link, when + duration, secondary person, checks).
 * Wired as the `spotlight` console panel type; these stories pass resolved props.
 */
const meta = {
  title: "Console/Spotlight",
  component: WidgetSpotlight,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    isLoading: { control: "boolean" },
    status: { control: "inline-radio", options: ["success", "running", "failed", "warning", "neutral"] },
  },
} satisfies Meta<typeof WidgetSpotlight>;

export default meta;
type Story = StoryObj<typeof meta>;

/** GitHub renders a square avatar at `https://github.com/<user>.png`. */
const avatar = (user: string) => `https://github.com/${user}.png`;

const MIN = 60 * 1000;
const HOUR = 60 * MIN;
/** A stable "N ago" ISO timestamp, computed once at import so stories don't drift. */
const ago = (ms: number) => new Date(Date.now() - ms).toISOString();

function SpotlightPanel({
  title,
  width = 560,
  height = 240,
  ...props
}: { title?: string; width?: number; height?: number } & ComponentProps<typeof WidgetSpotlight>) {
  return (
    <PanelFrame width={width} height={height}>
      <TypedPanelShell
        title={title}
        fallbackTitle="Spotlight"
        readOnly={false}
        onEdit={() => console.log("edit")}
        onDelete={() => console.log("delete")}
      >
        <WidgetSpotlight {...props} />
      </TypedPanelShell>
    </PanelFrame>
  );
}

/**
 * SuperPlane SaaS — the team's own delivery pipeline. A push to `launchpad`
 * ran the canvas end to end and shipped to production. The deploy stages become
 * the checks; the committer is a bot, so its avatar falls back to the icon.
 */
export const ProductionDeploy: Story = {
  render: (args) => <SpotlightPanel title="What's in production" {...args} />,
  args: {
    kicker: "Currently in production",
    status: "success",
    statusLabel: "Live",
    actor: { name: "cloud-robot" },
    title: "[production]: fix: refine run inspector input timeline behavior (#6010)",
    href: "https://github.com/superplanehq/launchpad/commit/da02113a7403a5015a0b16177ce6191fa9a0ac58",
    subtitle: "superplanehq/launchpad · main",
    timestamp: ago(18 * MIN),
    duration: 26 * 1000,
    checks: [
      { name: "SaaS CI", status: "success" },
      { name: "Build Image", status: "success" },
      { name: "Deploy Helm — Staging", status: "success" },
      { name: "Deploy Helm — Production", status: "success" },
      { name: "Promote — Production", status: "success" },
    ],
    isLoading: false,
  },
};

/** SuperPlane SaaS — a rollout in progress: staging is live, production is deploying. */
export const DeployInProgress: Story = {
  render: (args) => <SpotlightPanel title="What's in production" {...args} />,
  args: {
    kicker: "Rolling out to production",
    status: "running",
    statusLabel: "Deploying",
    actor: { name: "cloud-robot" },
    title: "[staging]: chore: bump helm chart values to 1.42.0",
    href: "https://github.com/superplanehq/launchpad/commits/main",
    subtitle: "superplanehq/launchpad · main",
    timestamp: ago(3 * MIN),
    duration: 2 * MIN + 40 * 1000,
    checks: [
      { name: "SaaS CI", status: "success" },
      { name: "Build Image", status: "success" },
      { name: "Deploy Helm — Staging", status: "success" },
      { name: "Deploy Helm — Production", status: "running" },
      { name: "Promote — Production", status: "neutral" },
    ],
    isLoading: false,
  },
};

/**
 * SuperPlane SaaS — a failed production deploy. Staging passed but the Helm
 * production step failed, so the pipeline fired "Discord: SaaS Prod Deployment
 * Failed" and never promoted.
 */
export const DeployFailed: Story = {
  render: (args) => <SpotlightPanel title="What's in production" {...args} />,
  args: {
    kicker: "Last production deploy failed",
    status: "failed",
    statusLabel: "Not promoted",
    actor: { name: "cloud-robot" },
    title: "[production]: chore: tighten helm resource limits",
    href: "https://github.com/superplanehq/launchpad/commits/main",
    subtitle: "superplanehq/launchpad · main",
    timestamp: ago(9 * MIN),
    duration: 1 * MIN + 12 * 1000,
    checks: [
      { name: "SaaS CI", status: "success" },
      { name: "Build Image", status: "success" },
      { name: "Deploy Helm — Staging", status: "success" },
      { name: "Deploy Helm — Production", status: "failed" },
      { name: "Promote — Production", status: "neutral" },
    ],
    isLoading: false,
  },
};

/**
 * Docs Reviewer — the PR docs-review agent. A pull request came in and the agent
 * decided the docs need updating, so the run failed the docs check. The review
 * steps become the checks; the secondary person is the agent itself.
 */
export const DocsReviewNeeded: Story = {
  render: (args) => <SpotlightPanel title="Latest docs review" {...args} />,
  args: {
    kicker: "Docs review",
    status: "failed",
    statusLabel: "Update needed",
    actor: { name: "forestileao", avatarUrl: avatar("forestileao") },
    title: "fix: refine run inspector input timeline behavior",
    href: "https://github.com/superplanehq/superplane/pull/6010",
    subtitle: "superplanehq/superplane · #6010",
    timestamp: ago(50 * MIN),
    duration: 3 * 1000,
    approver: { name: "SuperPlane Docs Agent", avatarUrl: avatar("superplanehq") },
    approverLabel: "Reviewed by",
    checks: [
      { name: "Skip if Draft", status: "success" },
      { name: "Mark Pending", status: "success" },
      { name: "Analyze Docs", status: "failed" },
      { name: "Create Docs Issue", status: "neutral" },
    ],
    isLoading: false,
  },
};

/** Docs Reviewer — a clean pass: the agent found no docs changes were required. */
export const DocsReviewApproved: Story = {
  render: (args) => <SpotlightPanel title="Latest docs review" {...args} />,
  args: {
    kicker: "Docs review",
    status: "success",
    statusLabel: "No changes needed",
    actor: { name: "felixgateru", avatarUrl: avatar("felixgateru") },
    title: "feat: add GitLab createMergeComment and addReaction components",
    href: "https://github.com/superplanehq/superplane/pull/6008",
    subtitle: "superplanehq/superplane · #6008",
    timestamp: ago(3 * HOUR),
    duration: 1 * 1000,
    approver: { name: "SuperPlane Docs Agent", avatarUrl: avatar("superplanehq") },
    approverLabel: "Reviewed by",
    checks: [
      { name: "Skip if Draft", status: "success" },
      { name: "Analyze Docs", status: "success" },
      { name: "Mark Success", status: "success" },
    ],
    isLoading: false,
  },
};

export const Loading: Story = {
  render: (args) => <SpotlightPanel title="What's in production" {...args} />,
  args: { isLoading: true },
};

export const Empty: Story = {
  render: (args) => <SpotlightPanel title="What's in production" {...args} />,
  args: { isLoading: false },
};
