import { describe, expect, it } from "vitest";

import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "./uniqueCanvasName";

describe("uniqueCanvasName", () => {
  it("returns the base name when it is free", () => {
    expect(uniqueCanvasName("Software Factory", ["Other App"])).toBe("Software Factory");
  });

  it("appends (2) when the base name is taken", () => {
    expect(uniqueCanvasName("Software Factory", ["Software Factory"])).toBe("Software Factory (2)");
  });

  it("increments past existing numeric suffixes", () => {
    expect(
      uniqueCanvasName("Software Factory", ["Software Factory", "Software Factory (2)", "Software Factory (3)"]),
    ).toBe("Software Factory (4)");
  });

  it("trims whitespace and falls back when base is empty", () => {
    expect(uniqueCanvasName("  ", ["App"])).toBe("App (2)");
    expect(uniqueCanvasName("  Software Factory  ", ["Software Factory"])).toBe("Software Factory (2)");
  });
});

describe("isCanvasNameAlreadyExistsError", () => {
  it("detects the API conflict message", () => {
    expect(isCanvasNameAlreadyExistsError(new Error("Canvas with the same name already exists"))).toBe(true);
    expect(
      isCanvasNameAlreadyExistsError({
        response: { data: { message: "Canvas with the same name already exists" } },
      }),
    ).toBe(true);
    expect(isCanvasNameAlreadyExistsError(new Error("something else"))).toBe(false);
  });
});
