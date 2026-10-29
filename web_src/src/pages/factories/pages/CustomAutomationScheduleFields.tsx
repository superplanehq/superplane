import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { CustomAutomationSchedule } from "../lib/customAutomation";

export function CustomAutomationScheduleFields({
  schedule,
  onChange,
}: {
  schedule: CustomAutomationSchedule;
  onChange: (schedule: CustomAutomationSchedule) => void;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <Label htmlFor="custom-automation-cron">Cron expression</Label>
      <Input
        id="custom-automation-cron"
        value={schedule.cronExpression}
        onChange={(event) => onChange({ ...schedule, cronExpression: event.target.value })}
        placeholder="0 9 * * *"
        className="font-mono"
        aria-describedby="custom-automation-cron-help"
        required
      />
      <p id="custom-automation-cron-help" className="text-xs text-muted-foreground">
        Use five fields, or six fields with seconds first. For weekdays at 09:00, use 0 9 * * MON-FRI.
      </p>
      <Label htmlFor="custom-automation-offset">UTC offset (hours)</Label>
      <Input
        id="custom-automation-offset"
        type="number"
        min={-12}
        max={14}
        step="any"
        value={Number.isFinite(schedule.utcOffset) ? schedule.utcOffset : ""}
        onChange={(event) => onChange({ ...schedule, utcOffset: event.target.valueAsNumber })}
        required
      />
      <p className="text-xs text-muted-foreground">
        Effective timezone: UTC{schedule.utcOffset >= 0 ? "+" : ""}
        {schedule.utcOffset}. This fixed offset does not change for daylight saving time.
      </p>
    </div>
  );
}
