import type React from "react";

import { getBackgroundColorClass } from "@/lib/colors";
import type { ExecutionInfo } from "../types";

export function Message({ lastExecution }: { lastExecution: ExecutionInfo | null }): React.ReactNode {
  if (!lastExecution) {
    return null;
  }

  const metadata = (lastExecution.metadata as Record<string, unknown> | null | undefined) ?? {};
  const message = (metadata["message"] as string | undefined) || "Empty message";
  const color = (metadata["color"] as string | undefined) || "gray";

  const colorClass = getBackgroundColorClass(color);

  return (
    <div className={`px-2 py-1.5 text-left text-base max-h-20 truncate ${colorClass}`}>
      <pre className="break-all whitespace-pre-wrap">{message}</pre>
    </div>
  );
}
