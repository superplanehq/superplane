import { LaneListenerList, type LaneListener } from "./LaneListenerList";
import { LINE_INTAKE_COPY, lineIntakeListenTitle, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

/** Intakes at the head of Backlog: each one opens the work orders below it. */
export function BacklogIntakeSources({
  intakes,
  showAddIntake = false,
  onOpenSettings,
  onAddIntake,
}: {
  intakes: ConfiguredLineIntakeSource[];
  showAddIntake?: boolean;
  onOpenSettings: (intake: ConfiguredLineIntakeSource) => void;
  onAddIntake?: () => void;
}) {
  const listeners: LaneListener[] = intakes.map((intake) => ({
    id: intake.intakeId,
    title: lineIntakeListenTitle(intake.source),
    iconSrc: intake.source.iconSrc,
    iconAlt: intake.source.iconAlt,
    healthy: intake.healthy,
    needsRepairLabel: LINE_INTAKE_COPY.needsRepair,
    settingsLabel: `Open ${intake.source.name} settings`,
    testId: `line-intake-source-${intake.intakeId}`,
    onOpenSettings: () => onOpenSettings(intake),
  }));

  return (
    <LaneListenerList
      listeners={listeners}
      testId="lines-backlog-intakes"
      addLabel="Add intake"
      addTestId="line-intake-add"
      onAdd={showAddIntake ? onAddIntake : undefined}
    />
  );
}
