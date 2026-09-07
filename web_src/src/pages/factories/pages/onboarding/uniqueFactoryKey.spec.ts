import { describe, expect, it, vi } from "vitest";

import { isFactoryKeyAlreadyExistsError, saveWithFreeWorkspaceKey } from "./uniqueFactoryKey";

const conflict = () => new Error("workspace key already exists in this organization");

describe("isFactoryKeyAlreadyExistsError", () => {
  it("detects the API conflict message", () => {
    expect(isFactoryKeyAlreadyExistsError(conflict())).toBe(true);
    expect(
      isFactoryKeyAlreadyExistsError({
        response: { data: { message: "workspace key already exists in this organization" } },
      }),
    ).toBe(true);
    expect(isFactoryKeyAlreadyExistsError(new Error("something else"))).toBe(false);
  });
});

describe("saveWithFreeWorkspaceKey", () => {
  it("saves the name-derived key when no workspace holds it", async () => {
    const save = vi.fn().mockResolvedValue("saved");

    await expect(saveWithFreeWorkspaceKey({ name: "Payments Service", save })).resolves.toBe("saved");
    expect(save).toHaveBeenCalledExactlyOnceWith("payme");
  });

  it("skips the keys the organization already holds", async () => {
    const save = vi.fn().mockResolvedValue("saved");

    await saveWithFreeWorkspaceKey({
      name: "Payments Service",
      takenKeys: ["payme", "payma"],
      save,
    });

    expect(save).toHaveBeenCalledExactlyOnceWith("paymb");
  });

  it("walks to the next variant while the API reports a conflict", async () => {
    const save = vi.fn().mockRejectedValueOnce(conflict()).mockRejectedValueOnce(conflict()).mockResolvedValue("saved");

    await expect(saveWithFreeWorkspaceKey({ name: "Payments Service", save })).resolves.toBe("saved");
    expect(save.mock.calls.map(([key]) => key)).toEqual(["payme", "payma", "paymb"]);
  });

  it("passes other failures to the caller", async () => {
    const save = vi.fn().mockRejectedValue(new Error("Network error"));

    await expect(saveWithFreeWorkspaceKey({ name: "Payments Service", save })).rejects.toThrow("Network error");
    expect(save).toHaveBeenCalledOnce();
  });
});
