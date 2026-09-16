import type { Meta, StoryObj } from "@storybook/react-vite";

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { PlanningSessionSurveyForm } from "../PlanningSessionSurveyForm";

const meta = {
  title: "Factories/Pages/Task Split Run/Refinement chat",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

function RefinementChatSurfaces() {
  return (
    <div className="mx-auto flex w-full max-w-lg flex-col gap-4">
      <Tabs defaultValue="description">
        <TabsList aria-label="Task views">
          <TabsTrigger value="description" className="sp-popup-view-tab">
            Task
          </TabsTrigger>
          <TabsTrigger value="log" className="sp-popup-view-tab">
            Automations
          </TabsTrigger>
        </TabsList>
      </Tabs>

      <div className="flex w-full justify-end">
        <div className="sp-user-note max-w-[92%] rounded-2xl border px-3.5 py-2.5">
          <div className="sp-user-note-label mb-1.5">Ada</div>
          <p className="text-[14px] leading-5 text-foreground">Keep the current dark theme.</p>
        </div>
      </div>

      <div className="flex w-full items-start px-2 py-0.5">
        <p className="text-[14px] leading-5 text-foreground">I will keep the current dark theme and continue.</p>
      </div>

      <div className="flex w-full justify-end">
        <div className="sp-survey-card max-w-[92%] rounded-2xl border px-3.5 py-3">
          <div className="sp-survey-accent text-[11px] font-medium">{CREATE_WITH_AGENT_COPY.answeredBy} Ada</div>
          <p className="mt-2.5 text-[12px] leading-4 text-muted-foreground">What should the agent build?</p>
          <p className="text-[14px] leading-5 font-medium text-foreground">Custom styled modal</p>
        </div>
      </div>

      <PlanningSessionSurveyForm
        survey={{ questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }] }}
        onSubmit={(text) => {
          console.log("survey submit", text);
        }}
      />
    </div>
  );
}

export const Light: Story = {
  name: "Light",
  render: () => <RefinementChatSurfaces />,
};

export const Dark: Story = {
  name: "Dark",
  globals: { theme: "dark" },
  parameters: {
    theme: "dark",
    backgrounds: { default: "dark" },
  },
  render: () => <RefinementChatSurfaces />,
};
