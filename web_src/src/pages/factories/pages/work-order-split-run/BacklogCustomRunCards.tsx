import type { ManualCustomAutomation } from "../../lib/customAutomation";
import { automationsFromStages, stageFromPhase } from "./redesign/automationsViewModel";
import { ConsoleAutomationCard } from "./redesign/consoleAutomationCard";
import type { SplitRunFixture } from "./splitRunMocks";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";

export function BacklogCustomRunCards({
  fixture,
  canUpdate,
  footerActions,
  manualCustomAutomations,
  ...identity
}: {
  fixture: SplitRunFixture;
  canUpdate: boolean;
  footerActions: SplitRunFooterActions;
  manualCustomAutomations?: readonly ManualCustomAutomation[];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
}) {
  const phases = fixture.phases.filter((phase) => phase.id.startsWith("backlog-custom-"));
  const automations = automationsFromStages(phases.map(stageFromPhase));
  if (automations.length === 0) return null;
  return (
    <section aria-label="Backlog automations" className="space-y-3 py-4">
      <h2 className="text-sm font-medium">Backlog automations</h2>
      {automations.map((automation) => (
        <ConsoleAutomationCard
          key={automation.id}
          automation={automation}
          phase={phases.find((phase) => phase.id === automation.latest.id)}
          phases={phases}
          {...identity}
          expandIdle={false}
          canStopRun={Boolean(canUpdate && identity.organizationId && identity.factoryId && identity.orderId)}
          actionBusy={footerActions.busy}
          onStopRun={(run) => void footerActions.handleStopAutomation(run)}
          manualRun={
            canUpdate ? manualCustomAutomations?.find((entry) => entry.id === automation.latest.appId) : undefined
          }
        />
      ))}
    </section>
  );
}
