let activeNoteId: string | null = null;
export const setActiveNoteId = (id: string | null) => {
  activeNoteId = id;
};

export const getActiveNoteId = () => activeNoteId;

export const restoreActiveNoteFocus = () => {
  if (typeof document === "undefined") return false;
  if (!activeNoteId) return false;
  const activeElement = document.activeElement;
  if (activeElement instanceof HTMLTextAreaElement && activeElement.dataset.noteId === activeNoteId) {
    return true;
  }
  const textarea = document.querySelector(`textarea[data-note-id="${activeNoteId}"]`);
  if (textarea instanceof HTMLTextAreaElement) {
    textarea.focus();
    return true;
  }
  return false;
};
