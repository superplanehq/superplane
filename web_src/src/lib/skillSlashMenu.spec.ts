import { describe, expect, it } from "bun:test";

import { isSkillSlashMenuTarget, skillSlashMenuHost, skillSlashMenuPortalFromRects } from "./skillSlashMenu";

describe("skillSlashMenuPortalFromRects", () => {
  it("places a compact menu below the caret inside the host", () => {
    expect(
      skillSlashMenuPortalFromRects({ left: 40, width: 480 }, { left: 56, bottom: 200 }, { left: 20, top: 80 }),
    ).toEqual({
      left: 36,
      width: 320,
      top: 124,
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
