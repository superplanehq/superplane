/** Factory Live has no run scope — hide global lastExecutions on the canvas. */
export function resolveShowNodeRuntimeActivity({
  showLiveActivity,
  factoryEmbed,
  isRunInspectionMode,
}: {
  showLiveActivity: boolean;
  factoryEmbed: boolean;
  isRunInspectionMode: boolean;
}): boolean {
  return showLiveActivity && (!factoryEmbed || isRunInspectionMode);
}
