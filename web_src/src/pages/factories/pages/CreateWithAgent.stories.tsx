import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { MemoryRouter } from "react-router";

import {
  CREATE_WITH_AGENT_DEMO_REPOSITORY,
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
        onSubmitSurvey={() => undefined}
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
        onSelectCreated={(order) => setView((current) => ({ ...current, right: { kind: "preview", order } }))}
        onRefineCreated={() => undefined}
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

export const OlderMessages: Story = {
  name: "Older messages",
  render: () => (
    <StaticSession
      initial={runningCreateWithAgentView({
        messages: Array.from({ length: 24 }, (_, index) => ({
          id: `line-${index}`,
          kind: "text" as const,
          role: index % 2 === 0 ? ("agent" as const) : ("user" as const),
          text:
            index % 2 === 0
              ? `Agent note ${index + 1}. The repository is ready.`
              : `User note ${index + 1}. Add a Size field to the checkout form.`,
        })),
      })}
    />
  ),
};

export const Survey: Story = {
  name: "Survey above the chat",
  render: () => (
    <StaticSession
      initial={waitingCreateWithAgentView({
        survey: {
          questions: [
            { prompt: "What is the priority?", options: ["High", "Low"] },
            { prompt: "What is the scope?", options: ["One file", "The service"] },
          ],
        },
      })}
    />
  ),
};

export const SessionTasks: Story = {
  name: "Tasks in this session",
  render: () => <StaticSession initial={runningCreateWithAgentView({ created: [createdOrder] })} />,
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
        right: { kind: "preview", order: createdOrder },
        endConfirmOpen: true,
      })}
    />
  ),
};
