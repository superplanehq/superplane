export function resolveFactoryAppCanvasSubtitle({
  description,
  factoryName,
}: {
  description?: string;
  factoryName?: string;
}) {
  if (description) {
    return description;
  }
  return `Canvas · ${factoryName?.trim() || "Workspace"}`;
}

export function resolveFactoryAppCanvasTitle(name?: string) {
  return name?.trim() || "App";
}

export function isFactoryAppConfigureMode(searchParams: URLSearchParams) {
  return searchParams.get("configure") === "1" || searchParams.get("edit") === "1";
}

export function isFactoryAppAgentPromptOpen(searchParams: URLSearchParams) {
  return searchParams.get("agentPrompt") === "1";
}

export function isFactoryAppYamlViewOpen(searchParams: URLSearchParams) {
  return searchParams.get("yaml") === "1";
}

export function isFactoryAppAgentPanelOpen(searchParams: URLSearchParams) {
  return searchParams.get("agent") === "1";
}

export function isFactoryAppComponentsOpen(searchParams: URLSearchParams) {
  return searchParams.get("blocks") === "1";
}

export function resolveFactoryLineName(
  lines: Array<{ id?: string; name?: string }> | undefined,
  lineId: string | null,
) {
  if (!lineId || !lines) {
    return null;
  }
  return lines.find((line) => line.id === lineId)?.name ?? null;
}
