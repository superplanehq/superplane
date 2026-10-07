export function formatAutomationLabel(nodeName: string | undefined, appName: string | undefined): string | undefined {
  if (nodeName && appName) {
    return `${nodeName} · ${appName}`;
  }
  return nodeName || appName;
}
