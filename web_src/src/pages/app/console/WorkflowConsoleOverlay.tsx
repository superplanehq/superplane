import { ConsoleOverlay, type ConsoleOverlayProps } from "./ConsoleOverlay";

type WorkflowConsoleOverlayProps = Omit<ConsoleOverlayProps, "readOnly" | "canImportYaml" | "canRunNodes"> & {
  isConsoleMode: boolean;
  canActOnCanvas: boolean;
  // Hides authoring affordances (panel edit/delete, drag/resize, YAML import)
  // when the app is in read mode.
  editLocked: boolean;
};

export function WorkflowConsoleOverlay({
  isConsoleMode,
  canActOnCanvas,
  editLocked,
  ...consoleProps
}: WorkflowConsoleOverlayProps) {
  if (!isConsoleMode) return null;

  // Runtime triggers (widget row actions, Node panel Run button) only depend
  // on permission/template state — not on edit mode. The in-flight run lock
  // handles disabling while a run is already executing.
  const runLocked = !canActOnCanvas;
  return (
    <ConsoleOverlay readOnly={editLocked} canImportYaml={!editLocked} canRunNodes={!runLocked} {...consoleProps} />
  );
}
