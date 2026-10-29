import { useId } from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

export function IntakeAutomaticIntakeSetting({
  paused,
  disabled,
  onPausedChange,
}: {
  paused: boolean;
  disabled: boolean;
  onPausedChange: (paused: boolean) => Promise<void> | void;
}) {
  const id = useId();
  return (
    <div className="flex items-start justify-between gap-6">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={id} className="text-sm">
          Automatic task creation
        </Label>
        <p id={`${id}-help`} className="text-xs text-muted-foreground">
          Changes apply immediately. Manual imports remain available.
        </p>
      </div>
      <Switch
        id={id}
        data-testid="intake-source-settings-automatic-intake"
        checked={!paused}
        disabled={disabled}
        aria-describedby={`${id}-help`}
        onCheckedChange={async (checked) => {
          try {
            await onPausedChange(!checked);
          } catch {
            // The host shows the error without changing the saved pause state.
          }
        }}
      />
    </div>
  );
}
