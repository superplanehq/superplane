import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { scheduleTimezoneChoices } from "@/lib/scheduleTimezones";
import type { CustomAutomationSchedule } from "../lib/customAutomation";

export function CustomAutomationScheduleFields({
  schedule,
  onChange,
  idPrefix = "custom-automation",
  nameSuffix,
  embedded = false,
}: {
  schedule: CustomAutomationSchedule;
  onChange: (schedule: CustomAutomationSchedule) => void;
  idPrefix?: string;
  nameSuffix?: string;
  embedded?: boolean;
}) {
  const suffix = nameSuffix ? ` ${nameSuffix}` : "";
  const timezoneLabel = `Timezone${suffix}`;
  return (
    <div className={embedded ? "space-y-3" : "space-y-3 rounded-lg border border-border p-4"}>
      <Label htmlFor={`${idPrefix}-cron`}>{`Cron expression${suffix}`}</Label>
      <Input
        id={`${idPrefix}-cron`}
        value={schedule.cronExpression}
        onChange={(event) => onChange({ ...schedule, cronExpression: event.target.value })}
        placeholder="0 9 * * *"
        className="font-mono"
        required
      />
      <Label htmlFor={`${idPrefix}-timezone`}>{timezoneLabel}</Label>
      <Select
        value={Number.isFinite(schedule.utcOffset) ? String(schedule.utcOffset) : ""}
        onValueChange={(value) => onChange({ ...schedule, utcOffset: Number(value) })}
      >
        <SelectTrigger id={`${idPrefix}-timezone`} className="w-full">
          <SelectValue placeholder="Choose a timezone" />
        </SelectTrigger>
        <SelectContent className="max-h-60">
          {scheduleTimezoneChoices(schedule.utcOffset).map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
