import { WorkOrderReactionBar } from "../../WorkOrderReactionBar";
import { CURRENT_REACTOR_NAME } from "./reactionMocks";
import { getMyReaction } from "./reactionModel";
import { useWorkOrderReactionsStore } from "./useWorkOrderReactionsStore";

/** Storybook-only reactions bar wired to the in-memory store, keyed by work order id. */
export function WorkOrderReactionsBar({ workOrderId }: { workOrderId: string }) {
  const { reactionsByWorkOrderId, toggleReaction } = useWorkOrderReactionsStore();
  const reactions = reactionsByWorkOrderId[workOrderId] ?? [];
  const myReaction = getMyReaction(reactions, CURRENT_REACTOR_NAME);

  return (
    <WorkOrderReactionBar
      reactions={reactions}
      myReaction={myReaction}
      onToggle={(emoji) => toggleReaction(workOrderId, emoji)}
    />
  );
}
