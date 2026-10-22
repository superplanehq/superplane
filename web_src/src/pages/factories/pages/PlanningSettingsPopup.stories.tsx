import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { planningReviewDraftFromCanvas } from "../lib/columnCanvasAgent";
import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { DEFAULT_PLANNING_SETTINGS } from "./planningSettingsModel";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryCanvas,
  factoryHistoryStoryCanvasId,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
} from "./automationRunHistory/factoryRunHistoryFixtures";

function PlanningPreview({
  paused = false,
  autoStart = false,
  loading = false,
  loadError = false,
  failSave = false,
  additionalInstructions = "",
}: {
  paused?: boolean;
  autoStart?: boolean;
  loading?: boolean;
  loadError?: boolean;
  failSave?: boolean;
  additionalInstructions?: string;
}) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [settings, setSettings] = useState({
    ...DEFAULT_PLANNING_SETTINGS,
    enabled: !paused,
    autoStartLineId: autoStart ? "main" : "",
  });
  const [agent, setAgent] = useState(() => {
    const draft = planningReviewDraftFromCanvas(factoryHistoryStoryCanvas("planning"), "refine-task")!;
    return {
      ...draft,
      components: draft.components.map((component) => ({
        ...component,
        configuration: { ...component.configuration, additionalInstructions },
      })),
    };
  });
  const [unavailable, setUnavailable] = useState(loading || loadError);
  const [shouldFailSave, setShouldFailSave] = useState(failSave);
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell initialPath="/?planningSettings=general" className="h-screen bg-background">
        <PlanningSettingsPopup
          title="Plans new tasks"
          organizationId={factoryHistoryStoryOrganization}
          factoryId={factoryHistoryStoryFactory}
          canvasId={factoryHistoryStoryCanvasId("planning")}
          settings={settings}
          lines={[{ id: "main", name: "Main" }]}
          onSave={(next) => {
            console.log("Save Planning settings", next);
            setSettings(next);
          }}
          onRetryAutomation={() => {
            console.log("Retry Planning configuration");
            setUnavailable(false);
          }}
          onClose={() => console.log("Close Planning settings")}
          agent={{
            draft: unavailable ? undefined : agent,
            isLoading: loading && unavailable,
            organizationId: factoryHistoryStoryOrganization,
            factoryId: factoryHistoryStoryFactory,
            onSave: (next) => {
              console.log("Save Planning agent settings", next);
              if (shouldFailSave) {
                setShouldFailSave(false);
                return Promise.reject(new Error("Could not save Planning settings. Try again."));
              }
              setAgent(next);
            },
          }}
        />
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Pages/Planning Settings",
  component: PlanningPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen", msw: { handlers: factoryHistoryHandlers("planning") } },
} satisfies Meta<typeof PlanningPreview>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Configuration: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(await canvas.findByTestId("field-model-hosted-model")).toBeVisible();
    await expect(canvas.queryByTestId("planning-settings-tab-agent")).not.toBeInTheDocument();
    await expect(canvas.queryByTestId("planning-review-resources")).not.toBeInTheDocument();
  },
};
export const WithInstructions: Story = {
  args: {
    additionalInstructions:
      "In our project, small features often touch both the UI and backend. That alone does not make them complex.",
  },
};
export const AutoStart: Story = { args: { autoStart: true } };
export const Paused: Story = { args: { paused: true, autoStart: true } };
export const Loading: Story = { args: { loading: true } };
export const LoadError: Story = { args: { loadError: true } };
export const SaveError: Story = {
  args: { failSave: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const instructions = await canvas.findByRole("textbox", { name: "Additional instructions" });
    await userEvent.type(instructions, "Preserve legacy annual billing plans.");
    await userEvent.click(canvas.getByTestId("planning-settings-save"));
    await expect(await canvas.findByRole("alert")).toBeVisible();
  },
};
