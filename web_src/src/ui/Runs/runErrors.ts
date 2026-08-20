export function normalizeRunErrors(errors: Array<string | undefined> | undefined | null): string[] {
  if (!errors || errors.length === 0) {
    return [];
  }

  return errors.flatMap((error) => {
    const message = error?.trim() ?? "";
    return message === "" ? [] : [message];
  });
}

export function shouldShowFactoryCanvasRunErrors({
  factoryEmbed,
  isRunInspectionMode,
  runInspectorOpen,
  errorCount,
}: {
  factoryEmbed?: boolean;
  isRunInspectionMode?: boolean;
  runInspectorOpen: boolean;
  errorCount: number;
}): boolean {
  return Boolean(factoryEmbed && isRunInspectionMode && !runInspectorOpen && errorCount > 0);
}
