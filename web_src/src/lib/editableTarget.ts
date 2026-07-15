/**
 * Editable elements where global keyboard shortcuts must stay inert so they
 * never eat the user's text input: plain inputs, textareas, selects,
 * contenteditable regions, and Monaco editors (code/expression fields).
 */
const EDITABLE_SELECTOR = 'input, textarea, select, [contenteditable="true"], .monaco-editor';

/**
 * Returns true when the event target sits inside an editable element, so a
 * global keydown handler can bail out before triggering a canvas-level action.
 */
export function isEditableTarget(target: EventTarget | null): boolean {
  return target instanceof Element && target.closest(EDITABLE_SELECTOR) !== null;
}
