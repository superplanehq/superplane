const MONACO_EDITOR_CHUNK_NAME = "monaco-editor";
export const REACT_DOM_CHUNK_NAME = "react-dom";

const EDITOR_CANCEL_FRAME_MARKERS = [MONACO_EDITOR_CHUNK_NAME, "@sentry", REACT_DOM_CHUNK_NAME];

export const editorCancelCodeSplittingGroups = [
  {
    name: MONACO_EDITOR_CHUNK_NAME,
    test: /monaco-editor/,
    includeDependenciesRecursively: false,
  },
  {
    name: REACT_DOM_CHUNK_NAME,
    test: /node_modules[/\\]react-dom[/\\]/,
    includeDependenciesRecursively: false,
  },
];

export function isEditorCancelFrame(filename: string | undefined): boolean {
  if (!filename) {
    return false;
  }

  return EDITOR_CANCEL_FRAME_MARKERS.some((marker) => filename.includes(marker));
}
