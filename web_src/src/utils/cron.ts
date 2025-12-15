/**
 * Simple cron parser for browser environment
 * Supports both 5-field and 6-field cron expressions
 */

export function getNextCronExecution(cronExpression: string, fromTime: Date): Date | null {
  try {
    const fields = cronExpression.trim().split(/\s+/);

    // Support both 5-field and 6-field formats
    let minute: string, hour: string, day: string, month: string, weekday: string;

    if (fields.length === 5) {
      // 5-field: minute hour day month weekday
      [minute, hour, day, month, weekday] = fields;
    } else if (fields.length === 6) {
      // 6-field: second minute hour day month weekday (ignore seconds for simplicity)
      [, minute, hour, day, month, weekday] = fields;
    } else {
      return null;
    }

    // Validate that weekday field doesn't contain spaces (common error)
    if (weekday.includes(" ")) {
      return null;
    }

    // Start from next minute
    const nextTime = new Date(fromTime);
    nextTime.setSeconds(0);
    nextTime.setMilliseconds(0);
    nextTime.setMinutes(nextTime.getMinutes() + 1);

    // Search for next valid time (limit search to avoid infinite loops)
    const maxIterations = 366 * 24 * 60; // 1 year worth of minutes
    let iterations = 0;

    while (iterations < maxIterations) {
      if (cronMatches(minute, hour, day, month, weekday, nextTime)) {
        return nextTime;
      }

      nextTime.setMinutes(nextTime.getMinutes() + 1);
      iterations++;
    }

    return null;
  } catch {
    return null;
  }
}

function cronMatches(minute: string, hour: string, day: string, month: string, weekday: string, date: Date): boolean {
  // Check minute (0-59)
  if (!matchesCronField(minute, date.getMinutes(), 0, 59)) return false;

  // Check hour (0-23)
  if (!matchesCronField(hour, date.getHours(), 0, 23)) return false;

  // Check month (1-12, but JS Date uses 0-11)
  if (!matchesCronField(month, date.getMonth() + 1, 1, 12)) return false;

  // Handle day of month vs day of week logic
  const dayMatches = matchesCronField(day, date.getDate(), 1, 31);
  const weekdayMatches = matchesCronField(weekday, date.getDay(), 0, 6);

  // If both day and weekday are wildcards, match
  if (day === "*" && weekday === "*") {
    return true;
  }
  // If only day is wildcard, check weekday
  else if (day === "*") {
    return weekdayMatches;
  }
  // If only weekday is wildcard, check day
  else if (weekday === "*") {
    return dayMatches;
  }
  // If both are specified, either can match (OR logic)
  else {
    return dayMatches || weekdayMatches;
  }
}

function matchesCronField(field: string, value: number, min: number, max: number): boolean {
  if (field === "*") return true;

  // Handle comma-separated values (1,2,3 or MON,TUE,WED)
  if (field.includes(",")) {
    return field.split(",").some((f) => matchesCronField(f.trim(), value, min, max));
  }

  // Handle step values (*/5, 2-10/3, MON-FRI/2)
  if (field.includes("/")) {
    const [range, step] = field.split("/");
    const stepNum = parseInt(step);

    if (range === "*") {
      return (value - min) % stepNum === 0;
    } else if (range.includes("-")) {
      const [start, end] = parseRange(range, min, max);
      if (start === -1 || end === -1) return false;
      return value >= start && value <= end && (value - start) % stepNum === 0;
    } else {
      const start = parseValue(range, min, max);
      if (start === -1) return false;
      return value >= start && (value - start) % stepNum === 0;
    }
  }

  // Handle ranges (2-5 or MON-FRI)
  if (field.includes("-")) {
    const [start, end] = parseRange(field, min, max);
    if (start === -1 || end === -1) return false;
    return value >= start && value <= end;
  }

  // Handle single values (including named days)
  const parsedValue = parseValue(field, min, max);
  return parsedValue !== -1 && parsedValue === value;
}

function parseValue(field: string, min: number, max: number): number {
  // Handle weekday names (only when min=0, max=6 for weekdays)
  if (min === 0 && max === 6) {
    const weekdayNames: Record<string, number> = {
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

    const upperField = field.toUpperCase();
    if (upperField in weekdayNames) {
      return weekdayNames[upperField];
    }
  }

  // Handle month names (only when min=1, max=12 for months)
  if (min === 1 && max === 12) {
    const monthNames: Record<string, number> = {
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

    const upperField = field.toUpperCase();
    if (upperField in monthNames) {
      return monthNames[upperField];
    }
  }

  // Handle numeric values
  const numValue = parseInt(field);
  if (isNaN(numValue)) return -1;

  return numValue >= min && numValue <= max ? numValue : -1;
}

function parseRange(range: string, min: number, max: number): [number, number] {
  const [startStr, endStr] = range.split("-");
  const start = parseValue(startStr, min, max);
  const end = parseValue(endStr, min, max);

  return [start, end];
}
