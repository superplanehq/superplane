import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";

export function IntakeSkipInitialImportField({
  checked,
  onCheckedChange,
  helper,
  testId,
  label = INTAKE_SKIP_INITIAL_IMPORT_COPY.label,
  disabled = false,
}: {
  /** True when SuperPlane imports existing items. */
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  helper: string;
  testId: string;
  label?: string;
  disabled?: boolean;
}) {
  const inputId = `${testId}-input`;
  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <Checkbox
        id={inputId}
        className="mt-0.5"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onCheckedChange(event.target.checked)}
        data-testid={testId}
      />
      <Label htmlFor={inputId} className="min-w-0 cursor-pointer flex-col items-start">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">{label}</span>
        <span className="mt-0.5 block text-[12px] font-normal text-muted-foreground">{helper}</span>
      </Label>
    </div>
  );
}
