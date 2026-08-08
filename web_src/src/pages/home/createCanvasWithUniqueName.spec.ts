import { describe, expect, it, vi } from "vitest";

import { createCanvasWithUniqueName, MAX_NAME_RETRY_ATTEMPTS } from "./createCanvasWithUniqueName";

const nameCollisionError = () => new Error("Canvas with the same name already exists");

describe("createCanvasWithUniqueName", () => {
  it("creates with the base name when it is free", async () => {
    const createCanvas = vi.fn().mockResolvedValue({ canvasId: "id-1" });

    const result = await createCanvasWithUniqueName({
      title: "Software Factory",
      existingNames: new Set(),
      createCanvas,
    });

    expect(result).toEqual({ canvasId: "id-1", canvasName: "Software Factory" });
    expect(createCanvas).toHaveBeenCalledTimes(1);
    expect(createCanvas).toHaveBeenCalledWith("Software Factory");
  });

  it("retries under a suffixed name when the create call reports a collision", async () => {
    const createCanvas = vi
      .fn()
      .mockRejectedValueOnce(nameCollisionError())
      .mockResolvedValueOnce({ canvasId: "id-2" });

    const result = await createCanvasWithUniqueName({
      title: "Software Factory",
      // Empty client-side cache: the collision is only discovered via the failed create call.
      existingNames: new Set(),
      createCanvas,
    });

    expect(result).toEqual({ canvasId: "id-2", canvasName: "Software Factory (2)" });
    expect(createCanvas).toHaveBeenNthCalledWith(1, "Software Factory");
    expect(createCanvas).toHaveBeenNthCalledWith(2, "Software Factory (2)");
  });

  it("propagates non-collision errors immediately, without retrying", async () => {
    const createCanvas = vi.fn().mockRejectedValue(new Error("network error"));

    await expect(
      createCanvasWithUniqueName({ title: "Software Factory", existingNames: new Set(), createCanvas }),
    ).rejects.toThrow("network error");
    expect(createCanvas).toHaveBeenCalledTimes(1);
  });

  it("throws once retries are exhausted", async () => {
    const createCanvas = vi.fn().mockRejectedValue(nameCollisionError());

    await expect(
      createCanvasWithUniqueName({
        title: "Software Factory",
        existingNames: new Set(),
        createCanvas,
        failureMessage: "Failed to create factory canvas",
      }),
    ).rejects.toThrow("Failed to create factory canvas");
    expect(createCanvas).toHaveBeenCalledTimes(MAX_NAME_RETRY_ATTEMPTS);
  });

  it("supports a custom collision predicate for callers whose API uses a different error shape", async () => {
    const conflictError = Object.assign(new Error("An App with the same name already exists"), { status: 409 });
    const createCanvas = vi.fn().mockRejectedValueOnce(conflictError).mockResolvedValueOnce({ canvasId: "id-3" });

    const result = await createCanvasWithUniqueName({
      title: "Slack Notifier",
      existingNames: new Set(),
      createCanvas,
      isNameCollisionError: (error) => error instanceof Error && (error as Error & { status?: number }).status === 409,
    });

    expect(result).toEqual({ canvasId: "id-3", canvasName: "Slack Notifier (2)" });
  });
});
