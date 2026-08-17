import type { WorkOrderReactionGroup } from "../../workOrderReactionTypes";

/** The emoji a given reactor currently has active, if any. */
export function getMyReaction(groups: WorkOrderReactionGroup[], reactorName: string): string | null {
  return groups.find((group) => group.reactorNames.includes(reactorName))?.emoji ?? null;
}

/**
 * Add, remove, or switch a reactor's reaction. Only one active reaction per
 * reactor at a time: picking a new emoji removes them from their previous
 * group and joins the new one; clicking their own emoji again removes it.
 */
export function toggleWorkOrderReaction(
  groups: WorkOrderReactionGroup[],
  emoji: string,
  reactorName: string,
): WorkOrderReactionGroup[] {
  const current = getMyReaction(groups, reactorName);

  const withoutMine = groups
    .map((group) => ({ ...group, reactorNames: group.reactorNames.filter((name) => name !== reactorName) }))
    .filter((group) => group.reactorNames.length > 0);

  if (current === emoji) {
    return withoutMine;
  }

  const existingGroup = withoutMine.find((group) => group.emoji === emoji);
  if (existingGroup) {
    return withoutMine.map((group) =>
      group.emoji === emoji ? { ...group, reactorNames: [...group.reactorNames, reactorName] } : group,
    );
  }

  return [...withoutMine, { emoji, reactorNames: [reactorName] }];
}
