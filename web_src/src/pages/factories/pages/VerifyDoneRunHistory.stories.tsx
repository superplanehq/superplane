import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { ColumnAutomationViewHost } from "./ColumnAutomationViewPopup";
import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { planningReviewDraftFromCanvas } from "../lib/columnCanvasAgent";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryCanvas,
  factoryHistoryStoryCanvasId,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
  factoryHistoryStoryNames,
  factoryHistoryStoryRunId,
  type VerificationStorySource,
} from "./automationRunHistory/factoryRunHistoryFixtures";

function VerificationPreview({ source, detail = false }: { source: VerificationStorySource; detail?: boolean }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const feedback = source === "discussion" || source === "checks";
  const parameter = feedback ? "prFeedbackSettings" : "automationTab";
  const canvasId = factoryHistoryStoryCanvasId(source);
  const agentId = factoryHistoryStoryCanvas(source).spec?.nodes?.find((node) => node.type === "TYPE_ACTION")?.id ?? "";
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell
        className="h-screen bg-background"
        initialPath={`/?${parameter}=runs${detail ? `&automationRun=${factoryHistoryStoryRunId}` : ""}`}
      >
        {feedback ? (
          <PRFeedbackSettingsPopup
            organizationId={factoryHistoryStoryOrganization}
            factoryId={factoryHistoryStoryFactory}
            canvasId={canvasId}
            settings={{
              source,
              name: factoryHistoryStoryNames[source],
              repository: "superplanehq/superplane",
              mention: "@superplaneagent",
              ignoreBots: true,
              addressAllBots: false,
              allowedBots: [],
              checkNames: ["CI"],
              maximumAttempts: 3,
              runnerIntegrationIds: [],
            }}
            healthy
            onSave={(next) => console.log("Save PR feedback", next)}
            onClose={() => console.log("Close feedback")}
            agent={{
              organizationId: factoryHistoryStoryOrganization,
              factoryId: factoryHistoryStoryFactory,
              automationId: canvasId,
              draft: planningReviewDraftFromCanvas(factoryHistoryStoryCanvas(source), agentId) ?? undefined,
              onSave: (next) => console.log("Save agent", next),
            }}
          />
        ) : (
          <ColumnAutomationViewHost
            organizationId={factoryHistoryStoryOrganization}
            factoryId={factoryHistoryStoryFactory}
            factoryKey="SUPER"
            columnKey={source === "pr-closure" || source === "custom-done" ? "done" : "verify"}
            canvasId={canvasId}
            title={factoryHistoryStoryNames[source]}
            onClose={() => console.log("Close automation")}
          />
        )}
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}
const meta = {
  title: "Factories/Pages/Automation run history/Verify and Done",
  component: VerificationPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof VerificationPreview>;
export default meta;
type Story = StoryObj<typeof meta>;

async function checkRun({ canvasElement }: { canvasElement: HTMLElement }) {
  const canvas = within(canvasElement.ownerDocument.body);
  await userEvent.click(await canvas.findByRole("button", { name: `Open run ${factoryHistoryStoryRunId}` }));
  await expect(await canvas.findByRole("button", { name: "Copy task ID SUPER-982" })).toBeVisible();
  await expect(canvas.queryByTestId("merge-confidence-config-canvas")).not.toBeInTheDocument();
  await userEvent.click(canvas.getByRole("button", { name: "Run history" }));
  await expect(await canvas.findByRole("table")).toBeVisible();
}
export const PRDiscussion: Story = {
  args: { source: "discussion" },
  parameters: { msw: { handlers: factoryHistoryHandlers("discussion") } },
  play: checkRun,
};
export const PRChecks: Story = {
  args: { source: "checks" },
  parameters: { msw: { handlers: factoryHistoryHandlers("checks") } },
  play: checkRun,
};
export const MergeConfidence: Story = {
  args: { source: "merge-confidence" },
  parameters: { msw: { handlers: factoryHistoryHandlers("merge-confidence") } },
  play: checkRun,
};
export const VisualEvidence: Story = {
  args: { source: "visual-evidence" },
  parameters: { msw: { handlers: factoryHistoryHandlers("visual-evidence") } },
  play: checkRun,
};
export const PRClosure: Story = {
  args: { source: "pr-closure" },
  parameters: { msw: { handlers: factoryHistoryHandlers("pr-closure") } },
  play: checkRun,
};
export const CustomVerify: Story = {
  args: { source: "custom-verify" },
  parameters: { msw: { handlers: factoryHistoryHandlers("custom-verify") } },
  play: checkRun,
};
export const CustomDone: Story = {
  args: { source: "custom-done" },
  parameters: { msw: { handlers: factoryHistoryHandlers("custom-done") } },
  play: checkRun,
};
export const RunPermalink: Story = {
  args: { source: "merge-confidence", detail: true },
  parameters: { msw: { handlers: factoryHistoryHandlers("merge-confidence") } },
};
export const FeatureDisabled: Story = {
  args: { source: "merge-confidence" },
  parameters: { msw: { handlers: factoryHistoryHandlers("merge-confidence", false) } },
};
