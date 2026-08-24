import { describe, expect, it, vi } from "vitest";

import { isFactoryNameAlreadyExistsError, saveWithFreeWorkspaceName } from "./uniqueFactoryName";

const conflict = () => new Error("factory with the same name already exists");

describe("isFactoryNameAlreadyExistsError", () => {
  it("detects the API conflict message", () => {
    expect(isFactoryNameAlreadyExistsError(conflict())).toBe(true);
    expect(
      isFactoryNameAlreadyExistsError({
        response: { data: { message: "factory with the same name already exists" } },
      }),
    ).toBe(true);
    expect(isFactoryNameAlreadyExistsError(new Error("something else"))).toBe(false);
  });
});

describe("saveWithFreeWorkspaceName", () => {
  it("saves the preferred name when no workspace holds it", async () => {
    const save = vi.fn().mockResolvedValue("saved");

    await expect(saveWithFreeWorkspaceName({ name: "Payments Service", save })).resolves.toBe("saved");
    expect(save).toHaveBeenCalledExactlyOnceWith("Payments Service");
  });

  it("skips the names the organization already holds", async () => {
    const save = vi.fn().mockResolvedValue("saved");

    await saveWithFreeWorkspaceName({
      name: "Payments Service",
      takenNames: ["Payments Service", "Payments Service 2"],
      save,
    });

    expect(save).toHaveBeenCalledExactlyOnceWith("Payments Service 3");
  });

  it("counts up the suffix while the API reports a conflict", async () => {
    const save = vi.fn().mockRejectedValueOnce(conflict()).mockRejectedValueOnce(conflict()).mockResolvedValue("saved");

    await expect(saveWithFreeWorkspaceName({ name: "Payments Service", save })).resolves.toBe("saved");
    expect(save.mock.calls.map(([name]) => name)).toEqual([
      "Payments Service",
      "Payments Service 2",
      "Payments Service 3",
    ]);
  });

  it("passes other failures to the caller", async () => {
    const save = vi.fn().mockRejectedValue(new Error("Network error"));

    await expect(saveWithFreeWorkspaceName({ name: "Payments Service", save })).rejects.toThrow("Network error");
    expect(save).toHaveBeenCalledOnce();
  });
});
