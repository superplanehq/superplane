import type { ConfigurationDisplayRow } from "./types";

export type ConfigurationDisplayBlock =
  | { type: "row"; row: ConfigurationDisplayRow }
  | { type: "group"; header: ConfigurationDisplayRow; children: ConfigurationDisplayBlock[] };

/** Structural headers for object fields, list containers, and list item groups. */
export function isNestedGroupHeader(row: ConfigurationDisplayRow): boolean {
  return row.key.endsWith(".__group") || row.key.endsWith(".__header");
}

export function parseConfigurationDisplayBlocks(rows: ConfigurationDisplayRow[]): ConfigurationDisplayBlock[] {
  const blocks: ConfigurationDisplayBlock[] = [];
  let index = 0;

  while (index < rows.length) {
    const row = rows[index];

    if (isNestedGroupHeader(row)) {
      const headerDepth = row.depth ?? 0;
      index += 1;
      const childRows: ConfigurationDisplayRow[] = [];

      while (index < rows.length && (rows[index].depth ?? 0) > headerDepth) {
        childRows.push(rows[index]);
        index += 1;
      }

      blocks.push({
        type: "group",
        header: row,
        children: parseConfigurationDisplayBlocks(childRows),
      });
      continue;
    }

    blocks.push({ type: "row", row });
    index += 1;
  }

  return blocks;
}
