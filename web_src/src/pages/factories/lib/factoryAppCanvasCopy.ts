export function resolveFactoryAppCanvasSubtitle({
  isConfigure,
  description,
  factoryName,
}: {
  isConfigure: boolean;
  description?: string;
  factoryName?: string;
}) {
  if (isConfigure) {
    return "Drag steps and reconnect edges to configure this automation.";
  }
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

export function resolveFactoryLineName(
  lines: Array<{ id?: string; name?: string }> | undefined,
  lineId: string | null,
) {
  if (!lineId || !lines) {
    return null;
  }
  return lines.find((line) => line.id === lineId)?.name ?? null;
}
