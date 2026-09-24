import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { ComponentStoryShell } from "../../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../../__fixtures__/factoriesStoryTheme";
import { WorkOrderStatusIcon } from "../../../workOrders/WorkOrderStatusIcon";
import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../../work-order-popup-redesign/popupShared";
import type { SplitRunFixture } from "../splitRunMocks";
import { SPLIT_RUN_SUPER503 } from "../splitRunSuper503Fixture";
import { displayStatusForLineStatus } from "../splitRunWorkOrderDisplay";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";
import { AutomationsTimelineVariant } from "./AutomationsTimelineVariant";

const STORY_CLIENT = new QueryClient({
  defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
});

/**
 * Redesign directions for the Automations tab, inside the popup chrome
 * and fed by SUPER-503. Compare with "Current design / Automations tab".
 */
const meta = {
  title: "Factories/Pages/Task Split Run/Redesign",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <QueryClientProvider client={STORY_CLIENT}>
        <ComponentStoryShell className="relative min-h-svh bg-muted/30 p-0">
          <Story />
        </ComponentStoryShell>
      </QueryClientProvider>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

/** Static copy of the popup's Task | Automations tabs, with Automations selected. */
function AutomationsTabsAccessory({ fixture }: { fixture: SplitRunFixture }) {
  return (
    <Tabs value="log">
      <TabsList aria-label="Task views">
        <TabsTrigger value="description" className="sp-popup-view-tab">
          Task
        </TabsTrigger>
        <TabsTrigger value="log" className="sp-popup-view-tab">
          <WorkOrderStatusIcon
            status={displayStatusForLineStatus(fixture.lineStatus)}
            title={fixture.lineStatus}
            className="size-3"
            aria-hidden
          />
          Automations
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}

function RedesignPopup({
  fixture,
  testId,
  children,
}: {
  fixture: SplitRunFixture;
  testId: string;
  children: ReactNode;
}) {
  return (
    <PopupShell testId={testId}>
      <PopupHeader title={fixture.title} accessory={<AutomationsTabsAccessory fixture={fixture} />}>
        <OwnerTimeCostRow
          fixture={fixture}
          usageByModel={fixture.usageByModel}
          usageByMachineType={fixture.usageByMachineType}
        />
      </PopupHeader>
      <div className="min-h-0 min-w-0 flex-1 overflow-y-auto px-5 py-4" data-testid={`${testId}-body`}>
        {children}
      </div>
    </PopupShell>
  );
}

/**
 * Stage timeline. One collapsible card per stage, outputs visible when
 * collapsed, transcript when open. PR runs nest under the pull request.
 */
export const StageTimeline: Story = {
  name: "Stage timeline",
  render: () => (
    <RedesignPopup fixture={SPLIT_RUN_SUPER503} testId="redesign-a-super503">
      <AutomationsTimelineVariant fixture={SPLIT_RUN_SUPER503} />
    </RedesignPopup>
  ),
};

/**
 * Run console. Step trace on the left with the same detailed transcript
 * as the stage timeline. Sticky summary on the right with status, spend,
 * outputs, and the action row.
 */
export const RunConsole: Story = {
  name: "Run console",
  render: () => (
    <RedesignPopup fixture={SPLIT_RUN_SUPER503} testId="redesign-b-super503">
      <AutomationsConsoleVariant fixture={SPLIT_RUN_SUPER503} />
    </RedesignPopup>
  ),
};
