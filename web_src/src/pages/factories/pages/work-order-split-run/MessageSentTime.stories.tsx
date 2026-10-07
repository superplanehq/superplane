import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { RequestMessage } from "./WorkOrderIntentRequest";
import { WorkOrderIntentTranscript } from "./WorkOrderIntentTranscript";

const sentAt = Date.UTC(2026, 7, 6, 10, 17);

const meta = {
  title: "Factories/Pages/Task Split Run/Message sent time",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="bg-background p-0">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const NarrowPane: Story = {
  name: "Narrow pane",
  render: () => (
    <div className="w-[14rem] overflow-hidden bg-background" data-testid="message-sent-time-pane">
      <div className="overflow-y-auto [scrollbar-gutter:stable]" data-testid="message-sent-time-scroll">
        <div className="px-4 py-3">
          <RequestMessage
            description="Show the next action on the empty billing page when this planning pane is narrow."
            createdAt={new Date(sentAt).toISOString()}
            asChat
          />
          <WorkOrderIntentTranscript
            organizationId="org-1"
            messages={[
              {
                id: "note-long",
                kind: "text",
                role: "user",
                text: "Keep the billing empty-state copy when this planning pane is narrow.",
                createdAtMs: sentAt,
              },
              {
                id: "note-short",
                kind: "text",
                role: "user",
                text: "Hi",
                createdAtMs: sentAt + 60_000,
              },
              {
                id: "survey-short",
                kind: "text",
                role: "user",
                origin: "survey",
                text: "Priority? High",
                createdAtMs: sentAt,
              },
              {
                id: "survey-1",
                kind: "text",
                role: "user",
                origin: "survey",
                text: "Scope? Keep the helper text beside the empty billing state",
                createdAtMs: sentAt,
              },
              {
                id: "agent-1",
                kind: "text",
                role: "agent",
                text: "Noted. The empty state stays in the planning chat.",
                createdAtMs: sentAt,
              },
              {
                id: "note-missing",
                kind: "text",
                role: "user",
                text: "No sent time.",
              },
              {
                id: "note-invalid",
                kind: "text",
                role: "user",
                text: "Invalid sent time.",
                createdAtMs: Number.NaN,
              },
              {
                id: "survey-missing",
                kind: "text",
                role: "user",
                origin: "survey",
                text: "Scope? No clock",
              },
            ]}
          />
        </div>
      </div>
    </div>
  ),
};
