import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { MemoryRouter } from "react-router";

import {
  CREATE_WITH_AGENT_DEMO_REPOSITORY,
  createWithAgentSurveyMessage,
  emptyCreateWithAgentView,
  runningCreateWithAgentView,
  waitingCreateWithAgentView,
} from "./createWithAgentDemo";
import { CreateWithAgentDialog } from "./CreateWithAgentDialog";
import type { CreateWithAgentView } from "./createWithAgentTypes";

const createdOrder = {
  id: "wo-1",
  key: "NEW-1",
  title: "Improve payments",
  description: "Draft from the agent session on acme/payments.",
};

const meta = {
  title: "Factories/Pages/Create with an Agent",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

function StaticSession({ initial }: { initial: CreateWithAgentView }) {
  const [view, setView] = useState(initial);
  return (
    <MemoryRouter>
      <CreateWithAgentDialog
        open
        workspaceName="Refunds"
        view={view}
        onComposerChange={(composer) => setView((current) => ({ ...current, composer }))}
        onSend={() => undefined}
        onAnswerSurvey={() => undefined}
        onDraftTitleChange={(title) =>
          setView((current) =>
            current.right.kind === "draft"
              ? { ...current, right: { kind: "draft", draft: { ...current.right.draft, title } } }
              : current,
          )
        }
        onDraftDescriptionChange={(description) =>
          setView((current) =>
            current.right.kind === "draft"
              ? { ...current, right: { kind: "draft", draft: { ...current.right.draft, description } } }
              : current,
          )
        }
        onCreateDraft={() => undefined}
        onSkipDraft={() => undefined}
        onWorkOnNew={() => undefined}
        onSelectCreated={() => undefined}
        onOpenCreated={() => undefined}
        onBackToList={() => undefined}
        onRequestClose={() => setView((current) => ({ ...current, endConfirmOpen: true }))}
        onCancelEnd={() => setView((current) => ({ ...current, endConfirmOpen: false }))}
        onConfirmEnd={() => undefined}
      />
    </MemoryRouter>
  );
}

export const Starting: Story = {
  name: "Machine starting",
  render: () => <StaticSession initial={emptyCreateWithAgentView()} />,
};

export const EmptyRight: Story = {
  name: "Empty right pane",
  render: () => <StaticSession initial={runningCreateWithAgentView()} />,
};

export const Waiting: Story = {
  name: "Waiting for you",
  render: () => <StaticSession initial={waitingCreateWithAgentView()} />,
};

export const Survey: Story = {
  name: "Survey",
  render: () => (
    <StaticSession
      initial={runningCreateWithAgentView({
        messages: [createWithAgentSurveyMessage()],
      })}
    />
  ),
};

export const Draft: Story = {
  name: "Draft on the right",
  render: () => (
    <StaticSession
      initial={runningCreateWithAgentView({
        right: {
          kind: "draft",
          draft: { title: "Improve payments", description: createdOrder.description },
        },
      })}
    />
  ),
};

export const SessionList: Story = {
  name: "Session list",
  render: () => (
    <StaticSession
      initial={runningCreateWithAgentView({
        created: [createdOrder],
        right: { kind: "list" },
      })}
    />
  ),
};

export const Preview: Story = {
  name: "Read-only preview",
  render: () => (
    <StaticSession
      initial={runningCreateWithAgentView({
        created: [createdOrder],
        right: { kind: "preview", order: createdOrder },
      })}
    />
  ),
};

export const EndConfirm: Story = {
  name: "End session confirm",
  render: () => (
    <StaticSession
      initial={runningCreateWithAgentView({
        repository: CREATE_WITH_AGENT_DEMO_REPOSITORY,
        created: [createdOrder],
        right: { kind: "list" },
        endConfirmOpen: true,
      })}
    />
  ),
};
