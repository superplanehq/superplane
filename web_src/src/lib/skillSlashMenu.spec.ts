import { describe, expect, it } from "bun:test";

import {
  isSkillSlashMenuTarget,
  skillSlashMenuHost,
  skillSlashMenuPortalFromRects,
  SKILL_SLASH_MENU_MAX_HEIGHT,
  SKILL_SLASH_MENU_OFFSET,
} from "./skillSlashMenu";

describe("skillSlashMenuPortalFromRects", () => {
  it("places a compact menu below the caret inside the host", () => {
    expect(
      skillSlashMenuPortalFromRects(
        { left: 40, width: 480 },
        { left: 56, top: 184, bottom: 200 },
        { left: 20, top: 80, bottom: 800 },
      ),
    ).toEqual({
      left: 36,
      width: 320,
      top: 124,
      maxHeight: SKILL_SLASH_MENU_MAX_HEIGHT,
    });
  });

  it("flips above the caret when the host cannot fit the menu below", () => {
    expect(
      skillSlashMenuPortalFromRects(
        { left: 40, width: 480 },
        { left: 56, top: 430, bottom: 448 },
        { left: 20, top: 80, bottom: 480 },
      ),
    ).toEqual({
      left: 36,
      width: 320,
      top: 430 - 80 - SKILL_SLASH_MENU_MAX_HEIGHT - SKILL_SLASH_MENU_OFFSET,
      maxHeight: SKILL_SLASH_MENU_MAX_HEIGHT,
    });
  });

  it("shrinks to the larger side when neither side fits a full list", () => {
    expect(
      skillSlashMenuPortalFromRects(
        { left: 40, width: 480 },
        { left: 56, top: 180, bottom: 198 },
        { left: 20, top: 0, bottom: 400 },
      ),
    ).toEqual({
      left: 36,
      width: 320,
      top: 202,
      maxHeight: 198,
    });
  });
});

describe("skillSlashMenuHost", () => {
  it("uses the create-task dialog when present", () => {
    const dialog = document.createElement("div");
    dialog.dataset.testid = "create-work-order-request-dialog";
    const editor = document.createElement("div");
    dialog.append(editor);
    document.body.append(dialog);

    expect(skillSlashMenuHost(editor)).toBe(dialog);
    dialog.remove();
  });
});

describe("isSkillSlashMenuTarget", () => {
  it("matches the skill slash menu", () => {
    const menu = document.createElement("ul");
    menu.dataset.testid = "skill-slash-menu";
    const option = document.createElement("button");
    menu.append(option);
    document.body.append(menu);

    expect(isSkillSlashMenuTarget(option)).toBe(true);
    expect(isSkillSlashMenuTarget(document.body)).toBe(false);

    menu.remove();
  });
});
