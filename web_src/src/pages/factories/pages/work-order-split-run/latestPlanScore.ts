import type { CreateWithAgentMessage } from "../createWithAgentTypes";

export function latestPlanScore(messages: CreateWithAgentMessage[]): number | undefined {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message.kind === "plan") {
      return message.score;
    }
  }
  return undefined;
}
