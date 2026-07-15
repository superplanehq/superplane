import type { ConsoleContextValue } from "./ConsoleContext";
import { resolveConsoleTrigger } from "./ConsoleContext";

/**
 * Resolve every persisted trigger reference (id-or-name) to its concrete
 * node id so the shared checkbox list can drive selection state off of
 * ids while YAML keeps the friendly name authors typed. Until the canvas
 * node catalog loads, unresolved entries stay represented by their raw
 * references so id-based selections remain toggleable without duplicates.
 */
export function resolveSelectedTriggerIds(
  triggers: readonly string[] | undefined,
  ctx: Pick<ConsoleContextValue, "nodes"> | undefined,
): string[] {
  if (!triggers || triggers.length === 0) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const reference of triggers) {
    const selectedId = resolveConsoleTrigger(ctx, reference)?.node.id ?? reference.trim();
    if (!selectedId || seen.has(selectedId)) continue;
    seen.add(selectedId);
    out.push(selectedId);
  }
  return out;
}

/**
 * Compute the next persisted `triggers` list after a checkbox toggle.
 *
 * Selection UI is id-based, but YAML should keep friendly names whenever
 * possible. Toggling off removes every persisted ref that resolves to the
 * given id (preserving unrelated / stale refs as written). Toggling on
 * appends the trigger's name when available, otherwise its id — never
 * rewriting the rest of the list to opaque ids.
 */
export function nextPersistedTriggerRefs(args: {
  triggers: readonly string[] | undefined;
  triggerId: string;
  selected: boolean;
  ctx: Pick<ConsoleContextValue, "nodes"> | undefined;
}): string[] | undefined {
  const { triggers, triggerId, selected, ctx } = args;
  const current = triggers ?? [];
  const referencesTrigger = (reference: string) =>
    reference.trim() === triggerId || resolveConsoleTrigger(ctx, reference)?.node.id === triggerId;

  if (selected) {
    const remaining = current.filter((reference) => !referencesTrigger(reference));
    return remaining.length > 0 ? remaining : undefined;
  }

  if (current.some(referencesTrigger)) {
    return current.length > 0 ? [...current] : undefined;
  }

  const name = resolveConsoleTrigger(ctx, triggerId)?.node.name?.trim();
  const persistAs = name || triggerId;
  return [...current, persistAs];
}
