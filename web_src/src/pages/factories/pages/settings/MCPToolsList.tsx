import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { useMemo, useState } from "react";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { enabledToolCount, mcpToolItems, sortMCPTools, type MCPToolSort } from "./mcpTools";

export function MCPToolsList({
  tools,
  isLoading,
  isError,
  disabledTools,
  lockedTools,
  canUpdate,
  onToggleTool,
  testId,
  initialSort = "name",
}: {
  tools: Array<{ name?: string; readOnly?: boolean }>;
  isLoading: boolean;
  isError: boolean;
  disabledTools: string[];
  lockedTools?: string[];
  canUpdate: boolean;
  onToggleTool: (toolName: string, enabled: boolean) => void;
  testId?: string;
  initialSort?: MCPToolSort;
}) {
  const [sort, setSort] = useState<MCPToolSort>(initialSort);
  const items = useMemo(() => sortMCPTools(mcpToolItems(tools), sort), [sort, tools]);
  const locked = new Set(lockedTools ?? []);

  if (isLoading) {
    return (
      <p className="px-1 py-2 text-[13px] text-muted-foreground" data-testid="mcp-tools-loading">
        {AGENT_RESOURCES_COPY.toolsLoading}
      </p>
    );
  }
  if (isError) {
    return (
      <p className="px-1 py-2 text-[13px] text-destructive" data-testid="mcp-tools-error">
        {AGENT_RESOURCES_COPY.toolsLoadError}
      </p>
    );
  }
  if (items.length === 0) {
    return (
      <p className="px-1 py-2 text-[13px] text-muted-foreground" data-testid="mcp-tools-empty">
        {AGENT_RESOURCES_COPY.toolsEmpty}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2" data-testid={testId ?? "mcp-tools-list"}>
      <div className="flex items-center justify-between gap-2">
        <p className="text-[12px] text-muted-foreground">
          {AGENT_RESOURCES_COPY.toolsCount(enabledToolCount(items, [...disabledTools, ...locked]), items.length)}
        </p>
        <Select value={sort} onValueChange={(value) => setSort(value as MCPToolSort)}>
          <SelectTrigger
            size="sm"
            className="w-36"
            aria-label={AGENT_RESOURCES_COPY.toolsSortLabel}
            data-testid="mcp-tools-sort"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="name">{AGENT_RESOURCES_COPY.toolsSortName}</SelectItem>
            <SelectItem value="read">{AGENT_RESOURCES_COPY.toolsSortRead}</SelectItem>
            <SelectItem value="write">{AGENT_RESOURCES_COPY.toolsSortWrite}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <ul className="divide-y divide-border">
        {items.map((tool) => {
          const workspaceOff = locked.has(tool.name);
          const checked = !workspaceOff && !disabledTools.includes(tool.name);
          return (
            <li key={tool.name} className="flex items-center gap-3 py-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[13px] font-medium text-foreground">{tool.name}</p>
                <p className={cn("text-[12px]", tool.readOnly ? "text-muted-foreground" : "text-muted-foreground")}>
                  {tool.readOnly ? AGENT_RESOURCES_COPY.toolRead : AGENT_RESOURCES_COPY.toolWrite}
                </p>
              </div>
              <Switch
                checked={checked}
                disabled={!canUpdate || workspaceOff}
                onCheckedChange={(enabled) => onToggleTool(tool.name, enabled)}
                aria-label={AGENT_RESOURCES_COPY.enableToolLabel(tool.name)}
                data-testid={`mcp-tool-enabled-${tool.name}`}
              />
            </li>
          );
        })}
      </ul>
    </div>
  );
}
