export type MCPToolSort = "name" | "read" | "write";

export type MCPToolItem = {
  name: string;
  readOnly: boolean;
};

export function mcpToolItems(tools: Array<{ name?: string; readOnly?: boolean }> | undefined): MCPToolItem[] {
  return (tools ?? []).flatMap((tool, index) => {
    const name = tool.name?.trim();
    if (!name) {
      return [];
    }
    return [{ name: name || `tool-${index + 1}`, readOnly: tool.readOnly === true }];
  });
}

export function sortMCPTools(tools: MCPToolItem[], sort: MCPToolSort): MCPToolItem[] {
  return [...tools].sort((left, right) => {
    if (sort === "read" && left.readOnly !== right.readOnly) {
      return left.readOnly ? -1 : 1;
    }
    if (sort === "write" && left.readOnly !== right.readOnly) {
      return left.readOnly ? 1 : -1;
    }
    return left.name.localeCompare(right.name);
  });
}

export function workspaceDisabledTools(resource: { disabledTools?: string[] | null }): string[] {
  return (resource.disabledTools ?? []).map((name) => name.trim()).filter(Boolean);
}

export function nextDisabledTools(disabled: string[], toolName: string, enabled: boolean): string[] {
  if (enabled) {
    return disabled.filter((name) => name !== toolName);
  }
  if (disabled.includes(toolName)) {
    return disabled;
  }
  return [...disabled, toolName];
}

export function enabledToolCount(tools: MCPToolItem[], disabled: string[]): number {
  if (tools.length === 0) {
    return 0;
  }
  const blocked = new Set(disabled);
  return tools.filter((tool) => !blocked.has(tool.name)).length;
}
