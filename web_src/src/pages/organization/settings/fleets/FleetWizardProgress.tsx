import { Check } from "lucide-react";

export function FleetWizardProgress({ steps, current }: { steps: readonly string[]; current: number }) {
  return (
    <ol aria-label="Setup steps" className="flex flex-wrap gap-x-5 gap-y-2 border-b border-border pb-4 text-[12px]">
      {steps.map((label, index) => (
        <li
          key={label}
          aria-current={index === current ? "step" : undefined}
          className={`flex items-center gap-2 ${index === current ? "font-medium text-foreground" : "text-muted-foreground"}`}
        >
          <span
            className={`flex size-5 items-center justify-center rounded-full ${index <= current ? "bg-foreground/10" : "bg-muted"}`}
          >
            {index < current ? <Check className="size-3" aria-hidden="true" /> : index + 1}
          </span>
          {label}
        </li>
      ))}
    </ol>
  );
}
