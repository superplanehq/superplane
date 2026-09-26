import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { connectionIsEstablished, connectionStatusLabel } from "./agentResourceDisplay";

export function ConnectionStatusDot({
  resource,
  className,
}: {
  resource: Parameters<typeof connectionIsEstablished>[0];
  className?: string;
}) {
  const connected = connectionIsEstablished(resource);
  const status = connectionStatusLabel(resource);
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={status}
          className={cn(
            "inline-block size-2 shrink-0 rounded-full border-0 p-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
            connected ? "bg-emerald-500" : "bg-muted-foreground/40",
            className,
          )}
          data-testid={connected ? "mcp-status-connected" : "mcp-status-disconnected"}
        />
      </TooltipTrigger>
      <TooltipContent side="top">{status}</TooltipContent>
    </Tooltip>
  );
}
