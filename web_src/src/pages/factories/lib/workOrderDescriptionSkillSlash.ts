import type { Editor } from "@tiptap/react";

import { skillSlashQueryAtCursor, type SkillSlashQuery } from "@/lib/skillSlash";

export type EditorSkillSlashQuery = SkillSlashQuery & { from: number };

export function skillSlashQueryInBlock(before: string, parentStart: number): EditorSkillSlashQuery | null {
  const query = skillSlashQueryAtCursor(before, before.length);
  if (!query) {
    return null;
  }
  return { ...query, from: parentStart + query.start };
}

export function skillSlashQueryInEditor(editor: Editor): EditorSkillSlashQuery | null {
  if (!editor.state.selection.empty) {
    return null;
  }
  const [before, parentStart] = skillSlashEditorBefore(editor);
  return skillSlashQueryInBlock(before, parentStart);
}

export function insertSkillCommandInEditor(editor: Editor, command: string): void {
  const { from } = editor.state.selection;
  const [before, parentStart] = skillSlashEditorBefore(editor);
  const query = skillSlashQueryInBlock(before, parentStart);
  const start = query ? query.from : from;
  editor.chain().focus().deleteRange({ from: start, to: from }).insertContent(`/${command} `).run();
}

export function applySkillSlashMenuKeyDown(
  event: KeyboardEvent,
  menu: {
    open: boolean;
    count: number;
    highlightIndex: number;
    onHighlight: (index: number) => void;
    onSelect: () => void;
    onDismiss: () => void;
  },
): boolean {
  if (!menu.open || menu.count === 0) {
    return false;
  }
  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
    return false;
  }
  if (event.key === "ArrowDown") {
    event.preventDefault();
    menu.onHighlight(menu.highlightIndex < menu.count - 1 ? menu.highlightIndex + 1 : 0);
    return true;
  }
  if (event.key === "ArrowUp") {
    event.preventDefault();
    menu.onHighlight(menu.highlightIndex > 0 ? menu.highlightIndex - 1 : menu.count - 1);
    return true;
  }
  if (event.key === "Enter" || event.key === "Tab") {
    event.preventDefault();
    menu.onSelect();
    return true;
  }
  if (event.key === "Escape") {
    event.preventDefault();
    menu.onDismiss();
    return true;
  }
  return false;
}

function skillSlashEditorBefore(editor: Editor): [string, number] {
  const $from = editor.state.selection.$from;
  return [$from.parent.textBetween(0, $from.parentOffset, undefined, "\0"), $from.start()];
}
