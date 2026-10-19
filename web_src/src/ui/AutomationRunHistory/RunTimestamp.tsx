function validDate(value: string | undefined) {
  if (!value || !Number.isFinite(Date.parse(value))) return null;
  return new Date(value);
}

function dayLabel(date: Date) {
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  if (date.toDateString() === today.toDateString()) return "Today";
  if (date.toDateString() === yesterday.toDateString()) return "Yesterday";
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    ...(date.getFullYear() !== today.getFullYear() ? { year: "numeric" } : {}),
  });
}

/** Local calendar day and absolute clock time. The tooltip keeps the full date and zone. */
export function RunTimestamp({ value, runStartedAt }: { value?: string; runStartedAt?: string }) {
  const date = validDate(value);
  if (!date) return <>—</>;
  const runDate = validDate(runStartedAt);
  const sameRunDay = runDate?.toDateString() === date.toDateString();
  const time = date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", hourCycle: "h23" });
  return (
    <time dateTime={value} title={date.toLocaleString(undefined, { timeZoneName: "short" })}>
      {sameRunDay ? time : `${dayLabel(date)}, ${time}`}
    </time>
  );
}
