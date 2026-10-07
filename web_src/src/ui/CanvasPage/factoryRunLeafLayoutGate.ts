/** True when the factory canvas should use ephemeral leaf-right run layout. */
export function shouldUseFactoryRunLeafLayout(input: {
  factoryEmbed: boolean;
  isRunInspectionMode: boolean;
  factoryDisplayLayout?: boolean;
}): boolean {
  if (!input.factoryEmbed) {
    return false;
  }
  return input.isRunInspectionMode || Boolean(input.factoryDisplayLayout);
}
