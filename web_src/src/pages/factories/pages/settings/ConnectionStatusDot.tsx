import { cn } from "@/lib/utils";

import { connectionIsEstablished } from "./agentResourceDisplay";

export function ConnectionStatusDot({
  resource,
  className,
}: {
  resource: Parameters<typeof connectionIsEstablished>[0];
  className?: string;
}) {
  const connected = connectionIsEstablished(resource);
  return (
    <span
      className={cn(
        "inline-block size-2 shrink-0 rounded-full",
        connected ? "bg-emerald-500" : "bg-muted-foreground/40",
        className,
      )}
      data-testid={connected ? "mcp-status-connected" : "mcp-status-disconnected"}
      aria-hidden
    />
  );
}
