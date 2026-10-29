import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { CustomAutomationSetupModal } from "./CustomAutomationSetupModal";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryCanvasId,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
  factoryHistoryStoryRunId,
} from "./automationRunHistory/factoryRunHistoryFixtures";

const automationId = factoryHistoryStoryCanvasId("custom-verify");
const api = `/api/v1/factories/${factoryHistoryStoryFactory}/custom-automations`;

function customAutomationHandlers({ empty = false, automatic = false } = {}) {
  return [
    http.get(`${api}/${automationId}`, () =>
      HttpResponse.json({
        customAutomation: {
          automation: { id: automationId, name: "Deploy preview" },
          settings: {
            startMode: automatic ? "START_MODE_AUTOMATIC" : "START_MODE_MANUAL",
            agent: { component: "runnerSuperPlane" },
            steps: [{ name: "Deploy preview", type: "TYPE_PROMPT", prompt: "Deploy a preview for this task." }],
          },
        },
      }),
    ),
    http.get(`/api/v1/canvases/${automationId}/runs`, () =>
      HttpResponse.json({
        runs: empty
          ? []
          : [
              {
                id: factoryHistoryStoryRunId,
                canvasId: automationId,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                createdAt: new Date(Date.now() - 125_000).toISOString(),
                finishedAt: new Date(Date.now() - 5_000).toISOString(),
                task: { id: "task-1", key: "SUPER-982", title: "Preview the new settings screen" },
              },
            ],
        totalCount: empty ? 0 : 1,
      }),
    ),
    ...factoryHistoryHandlers("custom-verify"),
  ];
}

function CustomAutomationPreview({ isNew = false, runs = false }: { isNew?: boolean; runs?: boolean }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell
        initialPath={`/?customAutomation=${isNew ? "new" : automationId}${runs ? "&automationTab=runs" : ""}`}
      >
        <FactoriesLayoutContext.Provider
          value={{
            organizationId: factoryHistoryStoryOrganization,
            factoryId: factoryHistoryStoryFactory,
            factoryKey: "SUPER",
            factory: null,
            factories: [],
            openCreateWorkOrder: () => console.log("Create task"),
          }}
        >
          <CustomAutomationSetupModal
            lineId="line"
            automationId={isNew ? undefined : automationId}
            onClose={() => console.log("Close automation")}
          />
        </FactoriesLayoutContext.Provider>
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Components/Run history/Custom automation configuration",
  component: CustomAutomationPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen", msw: { handlers: customAutomationHandlers() } },
} satisfies Meta<typeof CustomAutomationPreview>;
export default meta;
type Story = StoryObj<typeof meta>;

export const SavedConfiguration: Story = { args: {} };
export const SavedHistory: Story = {
  args: { runs: true },
  play: async ({ canvasElement }) => {
    const dialog = within(canvasElement.ownerDocument.body);
    await expect(await dialog.findByText("SUPER-982")).toBeVisible();
    await userEvent.click(dialog.getByRole("button", { name: `Open run ${factoryHistoryStoryRunId}` }));
    await userEvent.click(await dialog.findByRole("button", { name: "Run history" }));
    await expect(dialog.getByRole("table")).toBeVisible();
  },
};
export const AutomaticHistory: Story = {
  args: { runs: true },
  parameters: { msw: { handlers: customAutomationHandlers({ automatic: true }) } },
};
export const EmptyHistory: Story = {
  args: { runs: true },
  parameters: { msw: { handlers: customAutomationHandlers({ empty: true }) } },
};
export const NewConfiguration: Story = { args: { isNew: true } };
