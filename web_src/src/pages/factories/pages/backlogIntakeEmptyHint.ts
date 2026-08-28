export const BACKLOG_INTAKE_EMPTY_HINT = {
  linkLabel: "Intake",
  afterLink: " is running. Import an issue or create a task while you wait.",
} as const;

export function shouldShowBacklogIntakeEmptyHint({
  empty,
  hasIntake,
  onboarding,
}: {
  empty: boolean;
  hasIntake: boolean;
  onboarding: boolean;
}): boolean {
  return empty && hasIntake && !onboarding;
}
