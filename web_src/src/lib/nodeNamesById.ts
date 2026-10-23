/** Builds an id-to-name lookup from a list of items, skipping any missing either field. */
export function buildNodeNamesById<T>(
  items: T[],
  getId: (item: T) => string | undefined,
  getName: (item: T) => string | undefined,
): Record<string, string> {
  const names: Record<string, string> = {};
  for (const item of items) {
    const id = getId(item);
    const name = getName(item);
    if (id && name) {
      names[id] = name;
    }
  }
  return names;
}
