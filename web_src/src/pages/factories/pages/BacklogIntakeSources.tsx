import { LaneListenerList, type LaneListener } from "./LaneListenerList";
import { LINE_INTAKE_COPY, lineIntakeListenTitle, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

/** Intakes at the head of Backlog: each one opens the tasks below it. */
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
    title: lineIntakeListenTitle(intake.source, intake.paused),
    iconSrc: intake.source.iconSrc,
    iconAlt: intake.source.iconAlt,
    healthy: intake.healthy,
    paused: intake.paused,
    needsRepairLabel: LINE_INTAKE_COPY.needsRepair,
    pausedLabel: LINE_INTAKE_COPY.paused,
    pausedHelper: LINE_INTAKE_COPY.pausedHelper,
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
