const WORK_ORDER_DATE_TIME_FORMATTER = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});

export function formatWorkOrderDateTime(date: Date): string {
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return WORK_ORDER_DATE_TIME_FORMATTER.format(date).replace(",", "");
}
