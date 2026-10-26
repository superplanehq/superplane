import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";

export function IntakeSkipInitialImportField({
  checked,
  onCheckedChange,
  helper,
  testId,
  disabled,
}: {
  /** True when the initial import is skipped. */
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  helper: string;
  testId: string;
  disabled?: boolean;
}) {
  const inputId = `${testId}-input`;
  return (
    <div className="flex items-start gap-2 text-left">
      <Checkbox
        id={inputId}
        className="mt-0.5"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onCheckedChange(event.target.checked)}
        data-testid={testId}
      />
      <Label htmlFor={inputId} className="min-w-0 cursor-pointer flex-col items-start">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">
          {INTAKE_SKIP_INITIAL_IMPORT_COPY.label}
        </span>
        <span className="mt-0.5 block text-[12px] font-normal text-muted-foreground">{helper}</span>
      </Label>
    </div>
  );
}
