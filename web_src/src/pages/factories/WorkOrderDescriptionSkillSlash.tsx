import type { Editor } from "@tiptap/react";
import { useCallback, useEffect, useState, type MutableRefObject } from "react";

import { SkillSlashMenu } from "@/components/AgentSidebar/SkillSlashMenu";
import { useSkillSlashCandidates } from "@/hooks/useSkillSlashCandidates";
import type { SkillSlashCandidate } from "@/lib/skillSlash";
import { skillSlashMenuHost, skillSlashMenuPortalFromRects } from "@/lib/skillSlashMenu";

import {
  applySkillSlashMenuKeyDown,
  insertSkillCommandInEditor,
  skillSlashQueryInEditor,
} from "./lib/workOrderDescriptionSkillSlash";

export function WorkOrderDescriptionSkillSlash({
  editor,
  organizationId,
  factoryId,
  keyDownRef,
}: {
  editor: Editor;
  organizationId?: string;
  factoryId: string;
  keyDownRef: MutableRefObject<(event: KeyboardEvent) => boolean>;
}) {
  const [dismissed, setDismissed] = useState(false);
  const [highlightIndex, setHighlightIndex] = useState(0);
  const query = useEditorSkillQuery(editor);
  const open = Boolean(query) && !dismissed;
  const candidates = useSkillSlashCandidates(organizationId, factoryId, query?.query ?? "", open);
  const handleSelect = useCallback(
    (candidate: SkillSlashCandidate) => insertSkillCommandInEditor(editor, candidate.command),
    [editor],
  );

  useEffect(() => {
    setDismissed(false);
  }, [query?.start]);

  useEffect(() => {
    setHighlightIndex(0);
  }, [query?.query, query?.start]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent): boolean =>
      applySkillSlashMenuKeyDown(event, {
        open,
        count: candidates.length,
        highlightIndex,
        onHighlight: setHighlightIndex,
        onSelect: () => {
          const selected = candidates[highlightIndex] ?? candidates[0];
          if (selected) {
            handleSelect(selected);
          }
        },
        onDismiss: () => setDismissed(true),
      });
    keyDownRef.current = handleKeyDown;
    return () => {
      if (keyDownRef.current === handleKeyDown) {
        keyDownRef.current = () => false;
      }
    };
  }, [candidates, handleSelect, highlightIndex, keyDownRef, open]);

  const host = skillSlashMenuHost(editor.view.dom);
  return (
    <SkillSlashMenu
      candidates={open ? candidates : []}
      highlightIndex={highlightIndex}
      onHighlight={setHighlightIndex}
      onSelect={handleSelect}
      portal={{
        root: host,
        ...skillSlashMenuPortalFromRects(
          editor.view.dom.getBoundingClientRect(),
          editor.view.coordsAtPos(editor.state.selection.from),
          host.getBoundingClientRect(),
        ),
      }}
    />
  );
}

function useEditorSkillQuery(editor: Editor) {
  const [query, setQuery] = useState(() => skillSlashQueryInEditor(editor));

  useEffect(() => {
    const sync = () => setQuery(skillSlashQueryInEditor(editor));
    editor.on("update", sync);
    editor.on("selectionUpdate", sync);
    return () => {
      editor.off("update", sync);
      editor.off("selectionUpdate", sync);
    };
  }, [editor]);

  return query;
}
