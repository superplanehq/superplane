import { useEffect, useMemo, useState } from "react";
import { useFactorySchedulePreview } from "@/hooks/useFactoryData";

export function useScheduledIntakeSchedule(organizationId: string, factoryId: string, enabled: boolean) {
  const [mode, setMode] = useState("days");
  const [time, setTime] = useState("09:00");
  const [cronExpression, setCronExpression] = useState("0 9 * * *");
  const [timezone] = useState(() => ({
    offset: String(-new Date().getTimezoneOffset() / 60),
    name: Intl.DateTimeFormat().resolvedOptions().timeZone,
  }));
  const params = useMemo((): Record<string, string> => {
    if (mode === "cron") return { type: "cron", cronExpression, timezone: timezone.offset };
    const [hour, minute] = time.split(":");
    return { type: "days", hour, minute, timezone: timezone.offset };
  }, [mode, time, cronExpression, timezone.offset]);
  const [settled, setSettled] = useState<Record<string, string>>(params);
  useEffect(() => {
    const timer = setTimeout(() => setSettled(params), 300);
    return () => clearTimeout(timer);
  }, [params]);
  const valid = mode === "cron" ? Boolean(cronExpression.trim()) : /^([01]\d|2[0-3]):[0-5]\d$/.test(time);
  const preview = useFactorySchedulePreview(organizationId, factoryId, settled, enabled && valid);
  const checking =
    settled !== params || preview.isFetching || preview.isPending || (!preview.isFuture && !preview.error);
  return {
    mode,
    setMode,
    time,
    setTime,
    cronExpression,
    setCronExpression,
    timezone,
    params,
    nextRun: preview.isFuture ? preview.data : undefined,
    error: preview.error,
    retry: preview.refetch,
    checking,
    valid,
    ready: valid && !checking && preview.isSuccess && preview.isFuture,
  };
}
