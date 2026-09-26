import { Button } from "@/components/ui/button";
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
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={status}
          className={cn(
            "size-2 rounded-full p-0 hover:bg-transparent dark:hover:bg-transparent",
            connected ? "bg-emerald-500 hover:bg-emerald-500" : "bg-muted-foreground/40 hover:bg-muted-foreground/40",
            className,
          )}
          data-testid={connected ? "mcp-status-connected" : "mcp-status-disconnected"}
        />
      </TooltipTrigger>
      <TooltipContent side="top">{status}</TooltipContent>
    </Tooltip>
  );
}
