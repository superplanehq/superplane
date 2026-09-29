import { render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { SkillSlashMenu } from "./SkillSlashMenu";

const CANDIDATES = Array.from({ length: 8 }, (_, index) => ({
  id: String(index),
  command: `skill-${index}`,
  title: `Skill ${index}`,
  description: "",
}));

describe("SkillSlashMenu", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("scrolls the highlighted skill into view", () => {
    const scrollIntoView = vi.fn();
    vi.spyOn(HTMLElement.prototype, "scrollIntoView").mockImplementation(scrollIntoView);

    render(<SkillSlashMenu candidates={CANDIDATES} highlightIndex={7} onHighlight={vi.fn()} onSelect={vi.fn()} />);

    expect(scrollIntoView).toHaveBeenCalledWith({ block: "nearest" });
  });
});
