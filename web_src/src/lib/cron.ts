/**
 * Cron parser for the browser, kept in sync with the backend scheduler
 * (github.com/robfig/cron/v3) so the "next run" previewed in the UI matches the
 * time the backend will actually fire.
 *
 * Supports 5-field (minute hour day month weekday) and 6-field
 * (second minute hour day month weekday) expressions with `*`, `?`, ranges,
 * steps, lists, and month/weekday names.
 */

const MAX_SEARCH_YEARS = 5;

const WEEKDAY_NAMES: Record<string, number> = {
  SUN: 0,
  SUNDAY: 0,
  MON: 1,
  MONDAY: 1,
  TUE: 2,
  TUESDAY: 2,
  WED: 3,
  WEDNESDAY: 3,
  THU: 4,
  THURSDAY: 4,
  FRI: 5,
  FRIDAY: 5,
  SAT: 6,
  SATURDAY: 6,
};

const MONTH_NAMES: Record<string, number> = {
  JAN: 1,
  JANUARY: 1,
  FEB: 2,
  FEBRUARY: 2,
  MAR: 3,
  MARCH: 3,
  APR: 4,
  APRIL: 4,
  MAY: 5,
  JUN: 6,
  JUNE: 6,
  JUL: 7,
  JULY: 7,
  AUG: 8,
  AUGUST: 8,
  SEP: 9,
  SEPTEMBER: 9,
  OCT: 10,
  OCTOBER: 10,
  NOV: 11,
  NOVEMBER: 11,
  DEC: 12,
  DECEMBER: 12,
};

interface FieldSpec {
  min: number;
  max: number;
  names?: Record<string, number>;
}

const SECOND_FIELD: FieldSpec = { min: 0, max: 59 };
const MINUTE_FIELD: FieldSpec = { min: 0, max: 59 };
const HOUR_FIELD: FieldSpec = { min: 0, max: 23 };
const DAY_OF_MONTH_FIELD: FieldSpec = { min: 1, max: 31 };
const MONTH_FIELD: FieldSpec = { min: 1, max: 12, names: MONTH_NAMES };
const WEEKDAY_FIELD: FieldSpec = { min: 0, max: 6, names: WEEKDAY_NAMES };

interface ParsedField {
  /** Allowed values, ascending. */
  values: number[];
  /** True when the field was written as `*` or `?`, with or without a step. */
  isWildcard: boolean;
}

export interface CronSchedule {
  seconds: number[];
  minutes: number[];
  hours: number[];
  daysOfMonth: ParsedField;
  months: number[];
  daysOfWeek: ParsedField;
}

/**
 * Parse a cron expression into the values each field allows.
 * Returns null when the expression is not one the backend would accept.
 */
export function parseCronExpression(cronExpression: string): CronSchedule | null {
  const fields = cronExpression.trim().split(/\s+/).filter(Boolean);

  let second = "0";
  let minute: string, hour: string, day: string, month: string, weekday: string;

  if (fields.length === 5) {
    [minute, hour, day, month, weekday] = fields;
  } else if (fields.length === 6) {
    [second, minute, hour, day, month, weekday] = fields;
  } else {
    return null;
  }

  const seconds = parseField(second, SECOND_FIELD);
  const minutes = parseField(minute, MINUTE_FIELD);
  const hours = parseField(hour, HOUR_FIELD);
  const daysOfMonth = parseField(day, DAY_OF_MONTH_FIELD);
  const months = parseField(month, MONTH_FIELD);
  const daysOfWeek = parseField(weekday, WEEKDAY_FIELD);

  if (!seconds || !minutes || !hours || !daysOfMonth || !months || !daysOfWeek) {
    return null;
  }

  return {
    seconds: seconds.values,
    minutes: minutes.values,
    hours: hours.values,
    daysOfMonth,
    months: months.values,
    daysOfWeek,
  };
}

/**
 * The next time the expression fires strictly after `fromTime`, in local time.
 * Returns null for invalid expressions and for expressions with no occurrence
 * within the search horizon.
 */
export function getNextCronExecution(cronExpression: string, fromTime: Date): Date | null {
  const [next] = getNextCronExecutions(cronExpression, fromTime, 1);

  return next ?? null;
}

/**
 * The next `count` times the expression fires, ascending. Returns fewer entries
 * (possibly none) when the schedule has no further occurrence in the horizon.
 */
export function getNextCronExecutions(cronExpression: string, fromTime: Date, count: number): Date[] {
  const schedule = parseCronExpression(cronExpression);
  if (!schedule || count < 1 || Number.isNaN(fromTime.getTime())) {
    return [];
  }

  const executions: Date[] = [];
  let cursor = fromTime;

  for (let i = 0; i < count; i++) {
    const next = findNextExecution(schedule, cursor);
    if (!next) {
      break;
    }

    executions.push(next);
    cursor = next;
  }

  return executions;
}

function findNextExecution(schedule: CronSchedule, fromTime: Date): Date | null {
  // Occurrences are strictly after fromTime, at second granularity.
  const searchStart = new Date(fromTime.getTime());
  searchStart.setMilliseconds(0);
  searchStart.setSeconds(searchStart.getSeconds() + 1);

  const lastYear = fromTime.getFullYear() + MAX_SEARCH_YEARS;
  const day = new Date(searchStart.getFullYear(), searchStart.getMonth(), searchStart.getDate());
  let notBefore: Date | null = searchStart;

  while (day.getFullYear() <= lastYear) {
    if (matchesDay(schedule, day)) {
      const execution = firstExecutionOnDay(schedule, day, notBefore);
      if (execution) {
        return execution;
      }
    }

    notBefore = null;
    day.setDate(day.getDate() + 1);
    // Keep the cursor on midnight even across DST transitions.
    day.setHours(0, 0, 0, 0);
  }

  return null;
}

function matchesDay(schedule: CronSchedule, day: Date): boolean {
  if (!schedule.months.includes(day.getMonth() + 1)) {
    return false;
  }

  const dayOfMonthMatches = schedule.daysOfMonth.values.includes(day.getDate());
  const dayOfWeekMatches = schedule.daysOfWeek.values.includes(day.getDay());

  // Same rule as the backend: when either field is a wildcard both must match,
  // otherwise a match in either field is enough.
  if (schedule.daysOfMonth.isWildcard || schedule.daysOfWeek.isWildcard) {
    return dayOfMonthMatches && dayOfWeekMatches;
  }

  return dayOfMonthMatches || dayOfWeekMatches;
}

function firstExecutionOnDay(schedule: CronSchedule, day: Date, notBefore: Date | null): Date | null {
  const earliestHour = notBefore ? notBefore.getHours() : 0;
  const earliestMinute = notBefore ? notBefore.getMinutes() : 0;
  const earliestSecond = notBefore ? notBefore.getSeconds() : 0;

  for (const hour of schedule.hours) {
    if (hour < earliestHour) {
      continue;
    }

    for (const minute of schedule.minutes) {
      if (hour === earliestHour && minute < earliestMinute) {
        continue;
      }

      for (const second of schedule.seconds) {
        if (hour === earliestHour && minute === earliestMinute && second < earliestSecond) {
          continue;
        }

        return new Date(day.getFullYear(), day.getMonth(), day.getDate(), hour, minute, second, 0);
      }
    }
  }

  return null;
}

function parseField(field: string, spec: FieldSpec): ParsedField | null {
  const values = new Set<number>();
  let isWildcard = false;

  for (const entry of field.split(",")) {
    const parsed = parseFieldEntry(entry.trim(), spec);
    if (!parsed) {
      return null;
    }

    isWildcard = isWildcard || parsed.isWildcard;
    for (const value of parsed.values) {
      values.add(value);
    }
  }

  if (values.size === 0) {
    return null;
  }

  return { values: [...values].sort((a, b) => a - b), isWildcard };
}

function parseFieldEntry(entry: string, spec: FieldSpec): ParsedField | null {
  if (!entry) {
    return null;
  }

  const [rangePart, stepPart, ...rest] = entry.split("/");
  if (rest.length > 0) {
    return null;
  }

  let step = 1;
  if (stepPart !== undefined) {
    if (!/^\d+$/.test(stepPart)) {
      return null;
    }

    step = Number(stepPart);
    if (step < 1) {
      return null;
    }
  }

  const range = parseRange(rangePart, spec, stepPart !== undefined);
  if (!range) {
    return null;
  }

  const values: number[] = [];
  for (let value = range.start; value <= range.end; value += step) {
    values.push(value);
  }

  // Like the backend, a step larger than 1 makes the field restricted even when
  // it was written as `*` or `?`.
  return { values, isWildcard: range.isWildcard && step === 1 };
}

interface FieldRange {
  start: number;
  end: number;
  isWildcard: boolean;
}

function parseRange(rangePart: string, spec: FieldSpec, hasStep: boolean): FieldRange | null {
  if (rangePart === "*" || rangePart === "?") {
    return { start: spec.min, end: spec.max, isWildcard: true };
  }

  const bounds = rangePart.split("-");
  if (bounds.length > 2) {
    return null;
  }

  const start = parseValue(bounds[0], spec);
  if (start === null) {
    return null;
  }

  if (bounds.length === 1) {
    // `5/15` reads as "from 5 to the end of the range, every 15".
    return { start, end: hasStep ? spec.max : start, isWildcard: false };
  }

  const end = parseValue(bounds[1], spec);
  if (end === null || end < start) {
    return null;
  }

  return { start, end, isWildcard: false };
}

function parseValue(value: string, spec: FieldSpec): number | null {
  const named = spec.names?.[value.toUpperCase()];
  if (named !== undefined) {
    return named;
  }

  if (!/^\d+$/.test(value)) {
    return null;
  }

  const parsed = Number(value);
  if (parsed < spec.min || parsed > spec.max) {
    return null;
  }

  return parsed;
}
