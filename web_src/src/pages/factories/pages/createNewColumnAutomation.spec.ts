import { describe, expect, it, vi } from "vitest";

import { createNewColumnAutomationCanvas, NEW_COLUMN_AUTOMATION_NAME } from "./createNewColumnAutomation";

describe("createNewColumnAutomationCanvas", () => {
  it("creates a canvas named New Automation", async () => {
    const createCanvas = vi.fn().mockResolvedValue({ data: { canvas: { metadata: { id: "app-new" } } } });

    await expect(
      createNewColumnAutomationCanvas({
        factoryId: "factory-1",
        existingNames: ["Implement"],
        createCanvas,
      }),
    ).resolves.toBe("app-new");

    expect(createCanvas).toHaveBeenCalledWith({
      name: NEW_COLUMN_AUTOMATION_NAME,
      description: "",
      factoryId: "factory-1",
      method: "ui",
    });
  });

  it("retries with a numbered name when the name is already taken", async () => {
    const createCanvas = vi
      .fn()
      .mockRejectedValueOnce(new Error("Canvas with the same name already exists"))
      .mockResolvedValueOnce({ data: { canvas: { metadata: { id: "app-new-2" } } } });

    await expect(
      createNewColumnAutomationCanvas({
        factoryId: "factory-1",
        existingNames: [NEW_COLUMN_AUTOMATION_NAME],
        createCanvas,
      }),
    ).resolves.toBe("app-new-2");

    expect(createCanvas).toHaveBeenNthCalledWith(1, {
      name: `${NEW_COLUMN_AUTOMATION_NAME} (2)`,
      description: "",
      factoryId: "factory-1",
      method: "ui",
    });
    expect(createCanvas).toHaveBeenNthCalledWith(2, {
      name: `${NEW_COLUMN_AUTOMATION_NAME} (3)`,
      description: "",
      factoryId: "factory-1",
      method: "ui",
    });
  });
});
