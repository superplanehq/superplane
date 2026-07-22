import { ConsoleOverlay, type ConsoleOverlayProps } from "./ConsoleOverlay";

type WorkflowConsoleOverlayProps = Omit<
  ConsoleOverlayProps,
  "readOnly" | "canImportYaml" | "canRunNodes" | "runNodesDisabledReason"
> & {
  isConsoleMode: boolean;
  canActOnCanvas: boolean;
  editSessionUiReady: boolean;
  hasUncommittedCanvasDraftChanges: boolean;
  // Hides authoring affordances (panel edit/delete, drag/resize, YAML import)
  // when the app is in read mode.
  editLocked: boolean;
};

export function WorkflowConsoleOverlay({
  isConsoleMode,
  canActOnCanvas,
  editSessionUiReady,
  hasUncommittedCanvasDraftChanges,
  editLocked,
  ...consoleProps
}: WorkflowConsoleOverlayProps) {
  if (!isConsoleMode) return null;

  // Console-only edits do not affect workflow_nodes and remain safe to make
  // while invoking runtime actions. Canvas draft edits can make the rendered
  // node/template differ from the live node used by InvokeNodeTriggerHook.
  const draftStatusPending = consoleProps.showConsoleEditControls && !editSessionUiReady;
  const hasDraftLiveMismatch = draftStatusPending || hasUncommittedCanvasDraftChanges;
  const runLocked = !canActOnCanvas || hasDraftLiveMismatch;
  return (
    <ConsoleOverlay
      readOnly={editLocked}
      canImportYaml={!editLocked}
      canRunNodes={!runLocked}
      runNodesDisabledReason={canActOnCanvas && hasDraftLiveMismatch ? "uncommitted-canvas-changes" : undefined}
      {...consoleProps}
    />
  );
}
