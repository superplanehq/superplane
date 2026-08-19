export function listFieldItemTitle(item: unknown, index: number, itemLabel: string): string {
  if (item && typeof item === "object" && !Array.isArray(item)) {
    const record = item as Record<string, unknown>;
    const name = typeof record.name === "string" ? record.name.trim() : "";
    if (name) return name;

    const label = typeof record.label === "string" ? record.label.trim() : "";
    if (label) return label;

    const type = typeof record.type === "string" ? record.type.trim() : "";
    if (type) {
      const typeLabel = type.charAt(0).toUpperCase() + type.slice(1);
      // Generic container labels ("Step", "Item") read better as just the type.
      if (itemLabel === "Step" || itemLabel === "Item") {
        return typeLabel;
      }
      return `${itemLabel} (${typeLabel})`;
    }
  }

  return `${itemLabel} ${index + 1}`;
}
