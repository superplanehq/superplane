export const SKILL_SLASH_MENU_TEST_ID = "skill-slash-menu";

export type SkillSlashMenuPortal = {
  left: number;
  width: number;
  top: number;
  root: HTMLElement;
};

export function isSkillSlashMenuTarget(target: EventTarget | null): boolean {
  return target instanceof Element && Boolean(target.closest(`[data-testid="${SKILL_SLASH_MENU_TEST_ID}"]`));
}

export function skillSlashMenuHost(from: HTMLElement): HTMLElement {
  return (
    from.closest<HTMLElement>("[data-testid='create-work-order-request-dialog']") ??
    from.closest<HTMLElement>("[data-testid='create-work-order-dialog']") ??
    document.body
  );
}

export function skillSlashMenuPortalFromRects(
  box: { left: number; width: number },
  caret: { left: number; bottom: number },
  origin: { left: number; top: number },
): Omit<SkillSlashMenuPortal, "root"> {
  return {
    left: Math.max(12, caret.left - origin.left),
    width: Math.min(320, Math.max(0, box.width)),
    top: caret.bottom - origin.top + 4,
  };
}
