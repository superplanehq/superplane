import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { useScheduledIntakeSchedule } from "./useScheduledIntakeSchedule";
import { getApiErrorMessage } from "@/lib/errors";

type Schedule = ReturnType<typeof useScheduledIntakeSchedule>;

export function ScheduledIntakeScheduleFields({ schedule }: { schedule: Schedule }) {
  return (
    <div className="rounded-lg border border-border p-4 space-y-3">
      <div className="space-y-2">
        <p className="text-sm font-medium">Schedule</p>
        <Tabs value={schedule.mode} onValueChange={schedule.setMode}>
          <TabsList aria-label="Schedule type" className="grid w-full grid-cols-2">
            <TabsTrigger value="days">Daily</TabsTrigger>
            <TabsTrigger value="cron">Cron expression</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
      {schedule.mode === "cron" ? (
        <div className="space-y-2">
          <Label htmlFor="scheduled-intake-cron">Cron expression</Label>
          <Input
            id="scheduled-intake-cron"
            value={schedule.cronExpression}
            onChange={(event) => schedule.setCronExpression(event.target.value)}
            placeholder="0 9 * * *"
            className="font-mono"
            aria-describedby="scheduled-intake-cron-help"
            required
          />
          <p id="scheduled-intake-cron-help" className="text-xs text-muted-foreground">
            Use five fields, or six fields with seconds first. For weekdays at 09:00, use 0 9 * * MON-FRI.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          <Label htmlFor="scheduled-intake-time">Time</Label>
          <Input
            id="scheduled-intake-time"
            type="time"
            value={schedule.time}
            onChange={(event) => schedule.setTime(event.target.value)}
            required
          />
        </div>
      )}
      <p className="text-xs text-muted-foreground">Timezone: {schedule.timezone.name} (from your browser)</p>
      <SchedulePreview schedule={schedule} />
    </div>
  );
}

function SchedulePreview({ schedule }: { schedule: Schedule }) {
  if (!schedule.valid)
    return (
      <p role="status" className="text-sm text-muted-foreground">
        Enter a {schedule.mode === "cron" ? "cron expression" : "time"} to preview the next run.
      </p>
    );
  if (schedule.checking)
    return (
      <p role="status" className="text-sm text-muted-foreground">
        Checking next run…
      </p>
    );
  if (schedule.error)
    return (
      <div className="space-y-2">
        <p role="alert" className="text-sm text-destructive">
          {getApiErrorMessage(schedule.error, "Could not check the schedule. Try again.")}
        </p>
        <Button type="button" variant="outline" size="sm" onClick={() => void schedule.retry()}>
          Retry preview
        </Button>
      </div>
    );
  if (!schedule.nextRun) return null;
  return (
    <p role="status" className="text-sm text-muted-foreground">
      Next run:{" "}
      <time dateTime={schedule.nextRun}>
        {new Date(schedule.nextRun).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })}
      </time>
    </p>
  );
}
