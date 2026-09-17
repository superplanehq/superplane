import type { AgentActivity, AgentActivityItem, AgentActivityStatus, AgentToolItem } from "@/lib/agentActivity";
import { formatDuration } from "@/lib/duration";
import { commandDisplayText } from "@/pages/factories/pages/work-order-split-run/agentActivitySummary";

export function LiveLogAgentActivity({ activities }: { activities: AgentActivity[] }) {
  const items = activities.flatMap((activity) => activity.items);
  if (items.length === 0) {
    return null;
  }
  return (
    <div className="space-y-1">
      {items.map((item) => (
        <ActivityRow key={item.id} item={item} />
      ))}
    </div>
  );
}

function ActivityRow({ item }: { item: AgentActivityItem }) {
  if (item.type === "content") {
    const text = item.text.trim();
    if (!text) {
      return null;
    }
    return (
      <p
        className={
          item.kind === "reasoning" ? "whitespace-pre-wrap text-gray-600 dark:text-gray-400" : "whitespace-pre-wrap"
        }
      >
        {text}
      </p>
    );
  }
  if (item.type === "tool") {
    return <ToolRow tool={item} />;
  }
  return <p className="whitespace-pre-wrap text-gray-600 dark:text-gray-400">{item.text}</p>;
}

function ToolRow({ tool }: { tool: AgentToolItem }) {
  const command = commandDisplayText(tool.input) ?? tool.name;
  return (
    <p>
      <span className="font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{tool.kind}</span>{" "}
      <span>{command}</span>
      <span className="text-gray-500 dark:text-gray-400">
        {" "}
        {activityStatusLabel(tool.status)}
        {tool.durationMs != null ? ` ${formatDuration(tool.durationMs)}` : ""}
      </span>
    </p>
  );
}

function activityStatusLabel(status: AgentActivityStatus): string {
  switch (status) {
    case "passed":
      return "Passed";
    case "failed":
      return "Failed";
    case "cancelled":
      return "Cancelled";
    case "timed_out":
      return "Timed out";
    case "interrupted":
      return "Interrupted";
    default:
      return "Running";
  }
}
