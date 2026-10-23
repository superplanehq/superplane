import { INTAKE_OUTCOMES, type IntakeRunOutcome } from "./intakeRunOutcome";

export function IntakeOutcomeLabel({ outcome }: { outcome: IntakeRunOutcome }) {
  const { label, dot } = INTAKE_OUTCOMES[outcome];
  return (
    <span className="inline-flex items-center gap-2 whitespace-nowrap">
      <span aria-hidden className={`size-2 shrink-0 rounded-full ${dot}`} />
      {label}
    </span>
  );
}
