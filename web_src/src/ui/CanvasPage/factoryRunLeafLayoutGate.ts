/** Canvases keep saved node positions in edit and read-only modes. */
export function shouldUseFactoryRunLeafLayout(_input: {
  factoryEmbed: boolean;
  isRunInspectionMode: boolean;
  factoryDisplayLayout?: boolean;
}): boolean {
  return false;
}
