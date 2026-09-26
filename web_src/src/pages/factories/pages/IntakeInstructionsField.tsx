import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useId, type Dispatch, type SetStateAction } from "react";

import { INTAKE_INSTRUCTIONS_COPY, type IntakeSourceSettings } from "./intakeSourceSettingsModel";

/**
 * Text that SuperPlane adds to the end of every task this intake creates.
 * It is how one intake gives its tasks different guidance from another.
 */
export function IntakeInstructionsField({
  settings,
  onSettingsChange,
}: {
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
}) {
  const id = useId();

  return (
    <div className="min-w-0">
      <Label htmlFor={id} className="workspace-section-title">
        {INTAKE_INSTRUCTIONS_COPY.label}
      </Label>
      <Textarea
        id={id}
        className="mt-2 min-h-28"
        value={settings.instructions}
        placeholder={INTAKE_INSTRUCTIONS_COPY.placeholder}
        onChange={(event) => {
          const instructions = event.target.value;
          onSettingsChange((current) => ({ ...current, instructions }));
        }}
        data-testid="intake-instructions"
      />
      <p className="mt-2 text-[13px] leading-5 text-muted-foreground">{INTAKE_INSTRUCTIONS_COPY.help}</p>
    </div>
  );
}
