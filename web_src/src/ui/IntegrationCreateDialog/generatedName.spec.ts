import { describe, expect, it, vi } from "vitest";

import { createWithGeneratedName, isNameTakenError } from "./generatedName";

function nameTaken(name: string) {
  return { message: `an integration with the name ${name} already exists in this organization` };
}

describe("isNameTakenError", () => {
  it("recognizes the API conflict message", () => {
    expect(isNameTakenError(nameTaken("github-puppies-inc"))).toBe(true);
  });

  it("ignores other failures", () => {
    expect(isNameTakenError({ message: "permission denied" })).toBe(false);
    expect(isNameTakenError(undefined)).toBe(false);
  });
});

describe("createWithGeneratedName", () => {
  it("uses the base name when it is free", async () => {
    const create = vi.fn().mockResolvedValue("created");

    const { result, name } = await createWithGeneratedName({
      baseName: "github-puppies-inc",
      takenNames: new Set(),
      create,
    });

    expect(name).toBe("github-puppies-inc");
    expect(result).toBe("created");
    expect(create).toHaveBeenCalledExactlyOnceWith("github-puppies-inc");
  });

  it("skips names that are known to be taken", async () => {
    const create = vi.fn().mockResolvedValue("created");

    const { name } = await createWithGeneratedName({
      baseName: "github-puppies-inc",
      takenNames: new Set(["github-puppies-inc", "github-puppies-inc-2"]),
      create,
    });

    expect(name).toBe("github-puppies-inc-3");
  });

  it("retries with the next suffix when the API reports a conflict", async () => {
    const create = vi
      .fn()
      .mockRejectedValueOnce(nameTaken("github-puppies-inc"))
      .mockRejectedValueOnce(nameTaken("github-puppies-inc-2"))
      .mockResolvedValue("created");

    const { name } = await createWithGeneratedName({
      baseName: "github-puppies-inc",
      takenNames: new Set(),
      create,
    });

    expect(name).toBe("github-puppies-inc-3");
    expect(create).toHaveBeenCalledTimes(3);
  });

  it("reports failures that a new name cannot fix", async () => {
    const create = vi.fn().mockRejectedValue({ message: "permission denied" });

    await expect(
      createWithGeneratedName({ baseName: "github-puppies-inc", takenNames: new Set(), create }),
    ).rejects.toEqual({ message: "permission denied" });
    expect(create).toHaveBeenCalledTimes(1);
  });
});
