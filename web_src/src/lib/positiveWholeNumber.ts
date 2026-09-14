// Number.parseInt accepts a numeric prefix, so "1.5" and "12abc" become 1
// and 12. Reject the whole string instead, or a form saves a different
// value from the one the user typed.
export const parsePositiveWholeNumber = (value: string): number | null => {
  const trimmed = value.trim();
  if (!/^\d+$/.test(trimmed)) {
    return null;
  }

  const parsed = Number(trimmed);
  if (!Number.isSafeInteger(parsed) || parsed < 1) {
    return null;
  }

  return parsed;
};
