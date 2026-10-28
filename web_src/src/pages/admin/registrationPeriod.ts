export type RegistrationPreset = "today" | "week" | "month" | "7days" | "30days";
export type RegistrationPeriod = { start: string; end: string };

export function registrationPeriod(preset: RegistrationPreset, now = new Date()): RegistrationPeriod {
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const end = new Date(start);
  end.setDate(end.getDate() + 1);
  if (preset === "week") {
    start.setDate(start.getDate() - start.getDay());
    end.setTime(start.getTime());
    end.setDate(end.getDate() + 7);
  }
  if (preset === "month") {
    start.setDate(1);
    end.setTime(start.getTime());
    end.setMonth(end.getMonth() + 1);
  }
  if (preset === "7days") start.setDate(start.getDate() - 6);
  if (preset === "30days") start.setDate(start.getDate() - 29);
  return { start: start.toISOString(), end: end.toISOString() };
}
