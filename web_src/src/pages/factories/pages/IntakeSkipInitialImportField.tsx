import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";

export function IntakeSkipInitialImportField({
  checked,
  onCheckedChange,
  testId,
}: {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  testId: string;
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
        onChange={(event) => onCheckedChange(event.target.checked)}
        data-testid={testId}
      />
      <Label htmlFor={inputId} className="min-w-0 cursor-pointer flex-col items-start">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">
          {INTAKE_SKIP_INITIAL_IMPORT_COPY.label}
        </span>
        <span className="mt-0.5 block text-[12px] font-normal text-muted-foreground">
          {INTAKE_SKIP_INITIAL_IMPORT_COPY.helper}
        </span>
      </Label>
    </div>
  );
}
