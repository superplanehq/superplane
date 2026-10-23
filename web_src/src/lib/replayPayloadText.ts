export function stringifyReplayPayload(payload: unknown): string {
  if (payload === undefined || payload === null) {
    return "{}";
  }
  if (typeof payload === "string") {
    try {
      return JSON.stringify(JSON.parse(payload), null, 2);
    } catch {
      return payload;
    }
  }
  return JSON.stringify(payload, null, 2);
}
