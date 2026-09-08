import { describe, expect, it, vi } from "vitest";

import { NEW_COLUMN_AUTOMATION_NAME, createNewColumnAutomationCanvas } from "./createNewColumnAutomation";

describe("createNewColumnAutomationCanvas", () => {
  it("creates a canvas named New Automation", async () => {
    const createCanvas = vi.fn().mockResolvedValue({ data: { canvas: { metadata: { id: "app-new" } } } });

    await expect(
      createNewColumnAutomationCanvas({ factoryId: "fac-1", existingNames: [], createCanvas }),
    ).resolves.toBe("app-new");

    expect(createCanvas).toHaveBeenCalledWith({
      name: NEW_COLUMN_AUTOMATION_NAME,
      description: "",
      factoryId: "fac-1",
      method: "ui",
    });
  });

  it("uses a numbered name when New Automation is taken", async () => {
    const createCanvas = vi.fn().mockResolvedValue({ data: { canvas: { metadata: { id: "app-2" } } } });

    await createNewColumnAutomationCanvas({
      factoryId: "fac-1",
      existingNames: [NEW_COLUMN_AUTOMATION_NAME],
      createCanvas,
    });

    expect(createCanvas).toHaveBeenCalledWith(expect.objectContaining({ name: `${NEW_COLUMN_AUTOMATION_NAME} (2)` }));
  });

  it("retries when the API reports a name conflict", async () => {
    const createCanvas = vi
      .fn()
      .mockRejectedValueOnce(new Error("Canvas with the same name already exists"))
      .mockResolvedValueOnce({ data: { canvas: { metadata: { id: "app-2" } } } });

    await expect(
      createNewColumnAutomationCanvas({ factoryId: "fac-1", existingNames: [], createCanvas }),
    ).resolves.toBe("app-2");

    expect(createCanvas).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ name: `${NEW_COLUMN_AUTOMATION_NAME} (2)` }),
    );
  });

  it("rejects when the API returns no canvas id", async () => {
    const createCanvas = vi.fn().mockResolvedValue({});

    await expect(
      createNewColumnAutomationCanvas({ factoryId: "fac-1", existingNames: [], createCanvas }),
    ).rejects.toThrow("Failed to create automation");
  });
});
