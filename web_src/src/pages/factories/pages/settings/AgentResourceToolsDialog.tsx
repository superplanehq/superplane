import type { FactoriesFactoryAgentResource, FactoriesFactoryAgentResourceTool } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export function AgentResourceToolsDialog({
  open,
  resource,
  tools,
  isLoading,
  isError,
  onClose,
}: {
  open: boolean;
  resource?: FactoriesFactoryAgentResource;
  tools: FactoriesFactoryAgentResourceTool[];
  isLoading: boolean;
  isError: boolean;
  onClose: () => void;
}) {
  const name = resource?.name?.trim() || AGENT_RESOURCES_COPY.unnamedResource;

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent data-testid="agent-resource-tools-dialog">
        <DialogHeader>
          <DialogTitle>{name}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.toolsDialogDescription}</DialogDescription>
        </DialogHeader>
        <ToolsDialogBody tools={tools} isLoading={isLoading} isError={isError} />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {AGENT_RESOURCES_COPY.close}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ToolsDialogBody({
  tools,
  isLoading,
  isError,
}: {
  tools: FactoriesFactoryAgentResourceTool[];
  isLoading: boolean;
  isError: boolean;
}) {
  if (isLoading) {
    return (
      <p className="text-[13px] text-muted-foreground" data-testid="agent-resource-tools-loading">
        {AGENT_RESOURCES_COPY.toolsLoading}
      </p>
    );
  }
  if (isError) {
    return (
      <p className="text-[13px] text-destructive" data-testid="agent-resource-tools-error">
        {AGENT_RESOURCES_COPY.toolsLoadError}
      </p>
    );
  }
  if (tools.length === 0) {
    return (
      <p className="text-[13px] text-muted-foreground" data-testid="agent-resource-tools-empty">
        {AGENT_RESOURCES_COPY.toolsEmpty}
      </p>
    );
  }
  return (
    <ul className="max-h-80 divide-y divide-border overflow-y-auto" data-testid="agent-resource-tools-list">
      {tools.map((tool, index) => {
        const toolName = tool.name?.trim() || `tool-${index + 1}`;
        return (
          <li key={toolName} className="py-3 first:pt-0 last:pb-0">
            <p className="text-[13px] font-medium text-foreground">{toolName}</p>
            {tool.description?.trim() ? (
              <p className="mt-1 text-[12px] text-muted-foreground">{tool.description.trim()}</p>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}
