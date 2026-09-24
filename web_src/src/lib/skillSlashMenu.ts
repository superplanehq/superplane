export const SKILL_SLASH_MENU_TEST_ID = "skill-slash-menu";
export const SKILL_SLASH_MENU_MAX_HEIGHT = 224;
export const SKILL_SLASH_MENU_OFFSET = 4;

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
  caret: { left: number; top: number; bottom: number },
  origin: { left: number; top: number; bottom: number },
): Omit<SkillSlashMenuPortal, "root"> {
  const hostHeight = origin.bottom - origin.top;
  const maxTop = Math.max(SKILL_SLASH_MENU_OFFSET, hostHeight - SKILL_SLASH_MENU_MAX_HEIGHT - SKILL_SLASH_MENU_OFFSET);
  const below = caret.bottom - origin.top + SKILL_SLASH_MENU_OFFSET;
  const above = caret.top - origin.top - SKILL_SLASH_MENU_MAX_HEIGHT - SKILL_SLASH_MENU_OFFSET;
  const spaceBelow = origin.bottom - caret.bottom;
  const spaceAbove = caret.top - origin.top;
  const top =
    spaceBelow < SKILL_SLASH_MENU_MAX_HEIGHT + SKILL_SLASH_MENU_OFFSET && spaceAbove > spaceBelow ? above : below;
  return {
    left: Math.max(12, caret.left - origin.left),
    width: Math.min(320, Math.max(0, box.width)),
    top: Math.min(Math.max(top, SKILL_SLASH_MENU_OFFSET), maxTop),
  };
}
