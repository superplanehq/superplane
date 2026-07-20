import type React from "react";

import { getBackgroundColorClass } from "@/lib/colors";
import { withEventSectionDarkBackground } from "@/lib/eventSectionBackground";
import type { ExecutionInfo } from "../types";

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }

  return value as Record<string, unknown>;
}

export function Message({ lastExecution }: { lastExecution: ExecutionInfo | null }): React.ReactNode {
  if (!lastExecution) {
    return null;
  }

  const metadata = asRecord(lastExecution.metadata);
  const rawMessage = metadata["message"];
  const message = typeof rawMessage === "string" && rawMessage.length > 0 ? rawMessage : "Empty message";
  const rawColor = metadata["color"];
  const color = typeof rawColor === "string" && rawColor.length > 0 ? rawColor : "gray";

  const colorClass = withEventSectionDarkBackground(getBackgroundColorClass(color));

  return (
    <div className={`px-2 py-1.5 text-left text-sm max-h-20 truncate ${colorClass}`}>
      <pre className="break-all whitespace-pre-wrap">{message}</pre>
    </div>
  );
}
