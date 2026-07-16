export const timezoneOptions = [
  { label: "GMT-12 (Baker Island)", value: "-12" },
  { label: "GMT-11 (American Samoa)", value: "-11" },
  { label: "GMT-10 (Hawaii)", value: "-10" },
  { label: "GMT-9 (Alaska)", value: "-9" },
  { label: "GMT-8 (Los Angeles, Vancouver)", value: "-8" },
  { label: "GMT-7 (Denver, Phoenix)", value: "-7" },
  { label: "GMT-6 (Chicago, Mexico City)", value: "-6" },
  { label: "GMT-5 (New York, Toronto)", value: "-5" },
  { label: "GMT-4 (Santiago, Atlantic)", value: "-4" },
  { label: "GMT-3 (São Paulo, Buenos Aires)", value: "-3" },
  { label: "GMT-2 (South Georgia)", value: "-2" },
  { label: "GMT-1 (Azores)", value: "-1" },
  { label: "GMT+0 (London, Dublin, UTC)", value: "0" },
  { label: "GMT+1 (Paris, Berlin, Rome)", value: "1" },
  { label: "GMT+2 (Cairo, Helsinki, Athens)", value: "2" },
  { label: "GMT+3 (Moscow, Istanbul, Riyadh)", value: "3" },
  { label: "GMT+4 (Dubai, Baku)", value: "4" },
  { label: "GMT+5 (Karachi, Tashkent)", value: "5" },
  { label: "GMT+5:30 (Mumbai, Delhi)", value: "5.5" },
  { label: "GMT+6 (Dhaka, Almaty)", value: "6" },
  { label: "GMT+7 (Bangkok, Jakarta)", value: "7" },
  { label: "GMT+8 (Beijing, Singapore, Perth)", value: "8" },
  { label: "GMT+9 (Tokyo, Seoul)", value: "9" },
  { label: "GMT+9:30 (Adelaide)", value: "9.5" },
  { label: "GMT+10 (Sydney, Melbourne)", value: "10" },
  { label: "GMT+11 (Solomon Islands)", value: "11" },
  { label: "GMT+12 (Auckland, Fiji)", value: "12" },
  { label: "GMT+13 (Tonga, Samoa)", value: "13" },
  { label: "GMT+14 (Kiribati)", value: "14" },
];

export function getUserTimezoneOffset(): string {
  const offset = -new Date().getTimezoneOffset() / 60;
  return offset.toString();
}

export function resolveTimezoneDisplayValue(value: unknown): string {
  if (value === undefined || value === null || value === "current") {
    const userTimezone = getUserTimezoneOffset();
    return timezoneOptions.find((tz) => tz.value === userTimezone)?.value ?? "0";
  }

  return String(value);
}

export function resolveDefaultTimezoneValue(): string {
  const userTimezone = getUserTimezoneOffset();
  return timezoneOptions.find((tz) => tz.value === userTimezone) ? userTimezone : "0";
}
