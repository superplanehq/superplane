import type { ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { MarkdownBody, MarkdownBodyLoading } from "./MarkdownBody";
import { PanelFrame } from "./__stories__/storyDecorators";
import { prRiskReviewMarkdownBody, prRiskReviewMarkdownPanelSize } from "./__stories__/storyFixtures";

/**
 * Markdown panel content renderer.
 *
 * The real `MarkdownPanelCard` resolves its `{{ name.field }}` variables through
 * `useMarkdownVariables` (which hits canvas memory / run queries), so these
 * stories render the pure `MarkdownBody` with static `vars` instead. The card
 * uses a custom header (not `TypedPanelShell`), so `MarkdownCardFrame` mirrors
 * that chrome to keep the design faithful while focusing on body content.
 */
const meta = {
  title: "Console/Markdown",
  component: MarkdownBody,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
} satisfies Meta<typeof MarkdownBody>;

export default meta;
type Story = StoryObj<typeof meta>;

function MarkdownCardFrame({
  title,
  width,
  height,
  children,
}: {
  title: string;
  width?: number;
  height?: number;
  children: ReactNode;
}) {
  return (
    <PanelFrame width={width ?? 320} height={height ?? 320}>
      <div className="group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white">
        <div className="flex items-center justify-between rounded-t-lg py-1.5 pl-3 pr-1.5">
          <span className="truncate text-[13px] font-medium text-slate-700" title={title}>
            {title}
          </span>
        </div>
        <div className="min-h-0 flex-1 overflow-auto rounded-b-lg bg-white px-4 py-3">{children}</div>
      </div>
    </PanelFrame>
  );
}

const richBody = `# Deploy runbook

A quick reference for the **production** deploy flow.

## Steps

1. Merge to \`main\`
2. Wait for CI to pass
3. Approve the deploy gate

- Owner: _Platform team_
- Rollback: revert + redeploy

> Tip: keep an eye on the error rate panel after each deploy.

\`\`\`bash
make deploy ENV=prod
\`\`\`
`;

const tableBody = `## Service ownership

| Service | Owner | On-call |
| --- | --- | --- |
| api | Platform | @ada |
| web | Frontend | @grace |
| infra | SRE | @linus |
`;

const variableBody = `## Latest deploy

The current production version is **{{ deploy.version }}** and its status is \`{{ deploy.status }}\`.

Total successful builds today: {{ build.count }}.
`;

export const RichContent: Story = {
  render: () => (
    <MarkdownCardFrame title="Deploy runbook">
      <MarkdownBody body={richBody} vars={{}} />
    </MarkdownCardFrame>
  ),
};

export const Table: Story = {
  render: () => (
    <MarkdownCardFrame title="Service ownership">
      <MarkdownBody body={tableBody} vars={{}} />
    </MarkdownCardFrame>
  ),
};

export const WithInterpolatedVariables: Story = {
  render: () => (
    <MarkdownCardFrame title="Latest deploy">
      <MarkdownBody
        body={variableBody}
        vars={{
          deploy: { version: "v1.4.2", status: "passed" },
          build: { count: 42 },
        }}
      />
    </MarkdownCardFrame>
  ),
};

export const Loading: Story = {
  render: () => (
    <MarkdownCardFrame title="Latest deploy">
      <MarkdownBodyLoading />
    </MarkdownCardFrame>
  ),
};

/** Org fixture: `pr-risk-review` console → `readme` markdown panel with collapsible sections. */
export const PrRiskReviewReadme: Story = {
  render: () => (
    <MarkdownCardFrame
      title="PR Risk Review"
      width={prRiskReviewMarkdownPanelSize.width}
      height={prRiskReviewMarkdownPanelSize.height}
    >
      <MarkdownBody body={prRiskReviewMarkdownBody} vars={{}} />
    </MarkdownCardFrame>
  ),
};
