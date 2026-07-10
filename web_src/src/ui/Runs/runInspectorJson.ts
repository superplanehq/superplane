export function escapeJsonStringValue(value: string): string {
  const encoded = JSON.stringify(value);
  return encoded.slice(1, -1);
}
