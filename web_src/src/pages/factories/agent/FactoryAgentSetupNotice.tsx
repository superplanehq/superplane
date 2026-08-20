import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AgentSetupState } from "@/components/CanvasToolSidebar/agentSetupStateModel";
import { cn } from "@/lib/utils";
import { factoryAgentMessageClassName } from "./factoryAgentChrome";

export function FactoryAgentSetupNotice({
  firstName,
  onRetry,
  state,
}: {
  firstName: string;
  onRetry: () => void;
  state: AgentSetupState;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex-1 overflow-y-auto p-3">
        <div className="flex flex-col items-start">
          <div
            className={cn(
              "max-w-[85%] break-words rounded-lg bg-muted px-3 py-2 text-foreground",
              factoryAgentMessageClassName,
            )}
          >
            <AgentSetupMessage firstName={firstName} onRetry={onRetry} state={state} />
          </div>
        </div>
      </div>
    </div>
  );
}

function AgentSetupMessage({
  firstName,
  onRetry,
  state,
}: {
  firstName: string;
  onRetry: () => void;
  state: AgentSetupState;
}) {
  if (state === "unavailable") {
    return <>The SuperPlane agent isn't available on this instance.</>;
  }

  if (state === "failed") {
    return (
      <>
        I couldn't set up the SuperPlane agent. Try again in a moment.
        <div className="mt-3">
          <Button size="sm" variant="outline" onClick={onRetry}>
            Try again
          </Button>
        </div>
      </>
    );
  }

  return (
    <>
      Hi {firstName}! I'm your SuperPlane agent. Give me a moment to set up and I'll help you build.
      <div className="mt-2 flex items-center gap-2 text-[12px] text-muted-foreground">
        <Loader2 className="size-3 animate-spin" /> Setting up...
      </div>
    </>
  );
}
