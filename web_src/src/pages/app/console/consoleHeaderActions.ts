interface ConsoleHeaderActionsConfig {
  isEditing: boolean;
  isConsoleMode: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  openAddPanel: () => void;
  openYaml: () => void;
}

export function getConsoleHeaderActions({
  isEditing,
  isConsoleMode,
  canUpdateCanvas,
  canvasDeletedRemotely,
  openAddPanel,
  openYaml,
}: ConsoleHeaderActionsConfig) {
  const consoleVisible = isConsoleMode;
  const canEditConsole = isEditing && consoleVisible && canUpdateCanvas && !canvasDeletedRemotely;

  return {
    onConsoleAddPanel: canEditConsole ? openAddPanel : undefined,
    onConsoleOpenYaml: isEditing && consoleVisible ? openYaml : undefined,
    consoleYamlReadOnly: !canUpdateCanvas || canvasDeletedRemotely,
  };
}
