/**
 * Returns true for a normal left-click without modifier keys.
 * Use to distinguish "navigate in current tab" from "open in new tab" clicks on Links.
 */
export function isNormalClick(e: React.MouseEvent): boolean {
  return !e.metaKey && !e.ctrlKey && !e.shiftKey && !e.altKey && e.button === 0;
}
