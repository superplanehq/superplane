export const scheduleTimezones = [
  { label: "GMT-12 (Baker Island)", value: "-12" },
  { label: "GMT-11 (American Samoa)", value: "-11" },
  { label: "GMT-10 (Hawaii)", value: "-10" },
  { label: "GMT-9:30 (Marquesas Islands)", value: "-9.5" },
  { label: "GMT-9 (Alaska)", value: "-9" },
  { label: "GMT-8 (Los Angeles, Vancouver)", value: "-8" },
  { label: "GMT-7 (Denver, Phoenix)", value: "-7" },
  { label: "GMT-6 (Chicago, Mexico City)", value: "-6" },
  { label: "GMT-5 (New York, Toronto)", value: "-5" },
  { label: "GMT-4 (Santiago, Atlantic)", value: "-4" },
  { label: "GMT-3:30 (Newfoundland)", value: "-3.5" },
  { label: "GMT-3 (São Paulo, Buenos Aires)", value: "-3" },
  { label: "GMT-2:30 (Newfoundland summer)", value: "-2.5" },
  { label: "GMT-2 (South Georgia)", value: "-2" },
  { label: "GMT-1 (Azores)", value: "-1" },
  { label: "GMT+0 (London, Dublin, UTC)", value: "0" },
  { label: "GMT+1 (Paris, Berlin, Rome)", value: "1" },
  { label: "GMT+2 (Cairo, Helsinki, Athens)", value: "2" },
  { label: "GMT+3 (Moscow, Istanbul, Riyadh)", value: "3" },
  { label: "GMT+3:30 (Iran)", value: "3.5" },
  { label: "GMT+4 (Dubai, Baku)", value: "4" },
  { label: "GMT+4:30 (Afghanistan)", value: "4.5" },
  { label: "GMT+5 (Karachi, Tashkent)", value: "5" },
  { label: "GMT+5:30 (Mumbai, Delhi)", value: "5.5" },
  { label: "GMT+5:45 (Nepal)", value: "5.75" },
  { label: "GMT+6 (Dhaka, Almaty)", value: "6" },
  { label: "GMT+6:30 (Myanmar, Cocos Islands)", value: "6.5" },
  { label: "GMT+7 (Bangkok, Jakarta)", value: "7" },
  { label: "GMT+8 (Beijing, Singapore, Perth)", value: "8" },
  { label: "GMT+8:45 (Eucla)", value: "8.75" },
  { label: "GMT+9 (Tokyo, Seoul)", value: "9" },
  { label: "GMT+9:30 (Adelaide)", value: "9.5" },
  { label: "GMT+10 (Sydney, Melbourne)", value: "10" },
  { label: "GMT+10:30 (Lord Howe Island, Adelaide summer)", value: "10.5" },
  { label: "GMT+11 (Solomon Islands)", value: "11" },
  { label: "GMT+12 (Auckland, Fiji)", value: "12" },
  { label: "GMT+12:45 (Chatham Islands)", value: "12.75" },
  { label: "GMT+13 (Tonga, Samoa)", value: "13" },
  { label: "GMT+13:45 (Chatham Islands summer)", value: "13.75" },
  { label: "GMT+14 (Kiribati)", value: "14" },
];

export function scheduleTimezoneChoices(offset: number) {
  if (!Number.isFinite(offset)) {
    return scheduleTimezones;
  }
  const value = String(offset);
  if (scheduleTimezones.some((option) => option.value === value)) {
    return scheduleTimezones;
  }
  const sign = offset > 0 ? "+" : "";
  return [...scheduleTimezones, { label: `GMT${sign}${value}`, value }];
}
