import { describe, expect, it } from "vitest";

import { listFieldItemTitle } from "./listFieldItemTitle";

describe("listFieldItemTitle", () => {
  it("uses the item name when present", () => {
    expect(listFieldItemTitle({ name: "deploy", type: "string" }, 0, "Parameter")).toBe("deploy");
  });

  it("uses the item label when present", () => {
    expect(listFieldItemTitle({ label: "OpenAI", value: "openai" }, 0, "Option")).toBe("OpenAI");
  });

  it("prefers name over label when both are set", () => {
    expect(listFieldItemTitle({ name: "provider", label: "OpenAI" }, 0, "Option")).toBe("provider");
  });

  it("falls back to type and item label", () => {
    expect(listFieldItemTitle({ type: "boolean" }, 1, "Parameter")).toBe("Parameter (Boolean)");
  });

  it("falls back to numbered item label", () => {
    expect(listFieldItemTitle({}, 2, "Parameter")).toBe("Parameter 3");
  });
});
