interface ConsoleHeaderActionsConfig {
  isEditing: boolean;
  isConsoleMode: boolean;
  isTemplate: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  openAddPanel: () => void;
  openYaml: () => void;
}

export function getConsoleHeaderActions({
  isEditing,
  isConsoleMode,
  isTemplate,
  canUpdateCanvas,
  canvasDeletedRemotely,
  openAddPanel,
  openYaml,
}: ConsoleHeaderActionsConfig) {
  const consoleVisible = isConsoleMode;
  const canEditConsole = isEditing && consoleVisible && !isTemplate && canUpdateCanvas && !canvasDeletedRemotely;

  return {
    onConsoleAddPanel: canEditConsole ? openAddPanel : undefined,
    onConsoleOpenYaml: isEditing && consoleVisible ? openYaml : undefined,
    consoleYamlReadOnly: !canUpdateCanvas || isTemplate || canvasDeletedRemotely,
  };
}
