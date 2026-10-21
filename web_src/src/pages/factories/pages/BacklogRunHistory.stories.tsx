import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { IntakeSettingsHost } from "./IntakeSettingsHost";
import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { DEFAULT_PLANNING_SETTINGS } from "./planningSettingsModel";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, DEFAULT_SENTRY_INTAKE_SETTINGS } from "./intakeSourceSettingsModel";
import { lineIntakeSourceById } from "./lineIntakeModel";
import { planningReviewDraftFromCanvas } from "../lib/columnCanvasAgent";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryCanvas,
  factoryHistoryStoryCanvasId,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
  factoryHistoryStoryRunId,
  type BacklogStorySource,
} from "./automationRunHistory/factoryRunHistoryFixtures";

function BacklogPreview({ source, detail = false }: { source: BacklogStorySource; detail?: boolean }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const canvasId = factoryHistoryStoryCanvasId(source);
  const parameter = source === "planning" ? "planningSettings" : "settings";
  const initialPath = `/?${parameter}=runs${detail ? `&automationRun=${factoryHistoryStoryRunId}` : ""}`;
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell initialPath={initialPath} className="h-screen bg-background">
        {source === "planning" ? (
          <PlanningSettingsPopup
            organizationId={factoryHistoryStoryOrganization}
            factoryId={factoryHistoryStoryFactory}
            canvasId={canvasId}
            settings={DEFAULT_PLANNING_SETTINGS}
            onSave={(next) => console.log("Save Planning settings", next)}
            onClose={() => console.log("Close Planning")}
            agent={{
              draft: planningReviewDraftFromCanvas(factoryHistoryStoryCanvas(source), "refine-task") ?? undefined,
              organizationId: factoryHistoryStoryOrganization,
              factoryId: factoryHistoryStoryFactory,
              automationId: canvasId,
              onSave: (next) => console.log("Save Planning agent", next),
            }}
          />
        ) : (
          <IntakeSettingsHost
            organizationId={factoryHistoryStoryOrganization}
            factoryId={factoryHistoryStoryFactory}
            factoryKey="SUPER"
            repository="superplanehq/superplane"
            intake={{
              intakeId: source,
              appId: canvasId,
              integrationId: source === "sentry-exceptions" ? "sentry-connection" : undefined,
              resourceId: source === "sentry-exceptions" ? "payments" : undefined,
              source: lineIntakeSourceById(source)!,
              healthy: true,
              paused: false,
              settings:
                source === "sentry-exceptions"
                  ? { ...DEFAULT_SENTRY_INTAKE_SETTINGS, sentryProjectIds: ["payments"] }
                  : {
                      ...DEFAULT_GITHUB_INTAKE_SETTINGS,
                      name: source === "dependabot-alerts" ? "Dependabot alerts" : "GitHub issues",
                    },
            }}
            onClose={() => console.log("Close intake settings")}
          />
        )}
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Pages/Automation run history/Backlog",
  component: BacklogPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof BacklogPreview>;
export default meta;
type Story = StoryObj<typeof meta>;

async function checkHistory({ canvasElement }: { canvasElement: HTMLElement }) {
  const canvas = within(canvasElement.ownerDocument.body);
  const openRun = await canvas.findByRole("button", { name: `Open run ${factoryHistoryStoryRunId}` });
  await expect(canvas.queryByRole("tab", { name: "Automation" })).not.toBeInTheDocument();
  await userEvent.click(openRun);
  await userEvent.click(await canvas.findByRole("button", { name: /^(Evaluate incoming task|Plan task)/ }));
  await expect(await canvas.findByText(/Task evaluation complete/)).toBeVisible();
  await userEvent.click(canvas.getByRole("button", { name: "Run history" }));
  await expect(await canvas.findByRole("table")).toBeVisible();
}

export const GitHub: Story = {
  name: "GitHub",
  args: { source: "github-issues" },
  parameters: { msw: { handlers: factoryHistoryHandlers("github-issues") } },
  play: checkHistory,
};
export const Sentry: Story = {
  args: { source: "sentry-exceptions" },
  parameters: { msw: { handlers: factoryHistoryHandlers("sentry-exceptions") } },
  play: checkHistory,
};
export const Dependabot: Story = {
  args: { source: "dependabot-alerts" },
  parameters: { msw: { handlers: factoryHistoryHandlers("dependabot-alerts") } },
  play: checkHistory,
};
export const Planning: Story = {
  args: { source: "planning" },
  parameters: { msw: { handlers: factoryHistoryHandlers("planning") } },
  play: checkHistory,
};
export const RunPermalink: Story = {
  args: { source: "planning", detail: true },
  parameters: { msw: { handlers: factoryHistoryHandlers("planning") } },
};
export const FeatureDisabled: Story = {
  args: { source: "github-issues" },
  parameters: { msw: { handlers: factoryHistoryHandlers("github-issues", false) } },
};
