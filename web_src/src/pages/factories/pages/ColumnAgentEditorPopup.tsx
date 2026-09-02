import { PlanningReviewPopup } from "./PlanningReviewPopup";
import type { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";

type ColumnAgentEditor = ReturnType<typeof useColumnCanvasAgentEditor>;

export type ColumnAgentEditorPopupProps = {
  editor: ColumnAgentEditor;
  organizationId: string;
  automationHref?: string | null;
};

/** Renders the inline agent editor for a board column when its editor is open. */
export function ColumnAgentEditorPopup({ editor, organizationId, automationHref }: ColumnAgentEditorPopupProps) {
  if (!editor.editorOpen) {
    return null;
  }
  return (
    <PlanningReviewPopup
      key={editor.agentNode?.id ?? "agent"}
      onClose={editor.closeEditor}
      organizationId={organizationId}
      automationHref={automationHref ?? undefined}
      initialDraft={editor.draft ?? { title: "Editing Agent", components: [] }}
      isLoading={editor.isLoading || !editor.draft}
      onSave={editor.save}
    />
  );
}
