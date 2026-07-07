import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  approvalCanceledEvents,
  approvalEvents,
  cursorAgentEvents,
  githubErrorEvents,
  githubEvents,
  memoryEvents,
  runBashEvents,
} from "./storybooks/timelineGroupsFixtures";
import { EventTimeline, type RuntimeConfigNode, type TimelineEvent } from "./storybooks/timelineGroupsModel";

/**
 * Wireframe/design spec (never merged to production) for a flat run-step timeline:
 * a single feed of events on one rail, each rendered as a Card (GitHub-comment style)
 * or a Line (GitHub-commit style). No lifecycle grouping/nesting.
 */

function TimelineEventsPanel({ events, configNode }: { events: TimelineEvent[]; configNode?: RuntimeConfigNode }) {
  return (
    <div className="min-h-screen bg-white p-6">
      <div className="mx-auto w-full max-w-[480px] overflow-hidden rounded-lg border border-slate-200 shadow-sm">
        <EventTimeline events={events} configNode={configNode} />
      </div>
    </div>
  );
}

const meta = {
  title: "Runs Proto/Timeline Events (Wireframe)",
  component: TimelineEventsPanel,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof TimelineEventsPanel>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Approval: Story = {
  render: () => (
    <TimelineEventsPanel events={approvalEvents} configNode={{ component: "approval", name: "approve-deploy" }} />
  ),
};

export const ApprovalCanceled: Story = {
  render: () => (
    <TimelineEventsPanel
      events={approvalCanceledEvents}
      configNode={{ component: "approval", name: "approve-deploy" }}
    />
  ),
};

export const RunBash: Story = {
  render: () => <TimelineEventsPanel events={runBashEvents} configNode={{ component: "build", name: "run-tests" }} />,
};

export const CursorAgent: Story = {
  render: () => (
    <TimelineEventsPanel events={cursorAgentEvents} configNode={{ component: "cursor-agent", name: "cursor-agent" }} />
  ),
};

export const Github: Story = {
  render: () => <TimelineEventsPanel events={githubEvents} configNode={{ component: "github", name: "create-pr" }} />,
};

export const GithubError: Story = {
  render: () => (
    <TimelineEventsPanel events={githubErrorEvents} configNode={{ component: "github", name: "create-pr" }} />
  ),
};

export const Memory: Story = {
  render: () => (
    <TimelineEventsPanel events={memoryEvents} configNode={{ component: "memory", name: "upsert-memory" }} />
  ),
};
