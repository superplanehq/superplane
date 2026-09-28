import { describe, expect, it } from "bun:test";

import { duplicateWorkOrderCreateInput } from "./duplicateWorkOrderCreateInput";

describe("duplicateWorkOrderCreateInput", () => {
  it("copies the saved title and description words, drops file links, and omits assignees", () => {
    const source = {
      title: "Retry refunds",
      description: [
        "Refunds fail.",
        "",
        "![Checkout](sp-file://file-1)",
        "",
        'See <img src="sp-file://file-2" alt="shot"> and <a href="sp-file://file-3">notes</a> before [docs](https://example.com).',
        "",
        "[notes.md](sp-file://file-4)",
      ].join("\n"),
      assigneeIds: ["user-1"],
    };
    const input = duplicateWorkOrderCreateInput(source);

    expect(input.title).toBe("Retry refunds");
    expect(input.description).toBe("Refunds fail.\n\nSee and before [docs](https://example.com).");
    expect(input.description).not.toContain("sp-file://");
    expect(input).not.toHaveProperty("assigneeIds");
  });
});
