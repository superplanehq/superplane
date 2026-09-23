export function appendSpokenPhrase(current: string, phrase: string, maxLength?: number): string {
  const spoken = phrase.trim();
  if (!spoken) {
    return current;
  }
  const needsSpace = current.length > 0 && !/\s$/.test(current);
  const next = needsSpace ? `${current} ${spoken}` : `${current}${spoken}`;
  return maxLength === undefined ? next : next.slice(0, maxLength);
}

export function stripTrailingSpokenPhrase(value: string, phrase: string): string {
  const spoken = phrase.trim();
  if (!spoken) {
    return value;
  }
  if (value === spoken) {
    return "";
  }
  const spacedSuffix = ` ${spoken}`;
  if (value.endsWith(spacedSuffix)) {
    return value.slice(0, -spacedSuffix.length);
  }
  return value;
}
