/** Narrow unknown Grafana/JSON values to plain objects for safe property access. */
export function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }

  return value as Record<string, unknown>;
}

/**
 * Row count for a single Grafana data frame (`frame.data.values` — uses max column length).
 */
export function getFrameRowCount(frame: Record<string, unknown>): number {
  const data = asRecord(frame.data);
  const values = data?.values;
  if (!Array.isArray(values)) {
    return 0;
  }

  let maxLength = 0;
  for (const column of values) {
    if (Array.isArray(column) && column.length > maxLength) {
      maxLength = column.length;
    }
  }

  return maxLength;
}

/**
 * Total rows across all frames in a Grafana query API payload (`data.results.*.frames`).
 */
export function countGrafanaQueryResponseRows(responseData: Record<string, unknown>): number {
  const results = responseData.results;
  if (!results || typeof results !== "object" || Array.isArray(results)) {
    return 0;
  }

  let total = 0;
  for (const refId of Object.keys(results)) {
    const result = asRecord((results as Record<string, unknown>)[refId]);
    if (!result) {
      continue;
    }

    const frames = Array.isArray(result.frames) ? result.frames : [];
    for (const frameValue of frames) {
      const frame = asRecord(frameValue);
      if (!frame) {
        continue;
      }
      total += getFrameRowCount(frame);
    }
  }

  return total;
}
