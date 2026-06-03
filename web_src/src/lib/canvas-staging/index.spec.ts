import "fake-indexeddb/auto";
import { beforeEach, describe, expect, it } from "vitest";

import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./types";
import { clearStaging, getStaging, hasStagingFiles, openStagingDB, putStaging } from "./index";

describe("canvas-staging", () => {
  beforeEach(async () => {
    const db = await openStagingDB();
    db.close();
    await clearStaging("canvas-1", "drafts/user-1");
  });

  it("stores and reads staged files per canvas and branch", async () => {
    await putStaging({
      canvasId: "canvas-1",
      branch: "drafts/user-1",
      baseHeadSha: "abc1234567890123456789012345678901234567890",
      files: {
        [CANVAS_YAML_PATH]: "name: test",
        [CONSOLE_YAML_PATH]: "panels: []",
      },
      updatedAt: Date.now(),
    });

    const record = await getStaging("canvas-1", "drafts/user-1");

    expect(record).not.toBeNull();
    expect(record?.files[CANVAS_YAML_PATH]).toBe("name: test");
    expect(record?.files[CONSOLE_YAML_PATH]).toBe("panels: []");
    expect(record?.baseHeadSha).toBe("abc1234567890123456789012345678901234567890");
  });

  it("returns null when no staging exists", async () => {
    expect(await getStaging("canvas-1", "drafts/missing")).toBeNull();
  });

  it("clears staging for a branch", async () => {
    await putStaging({
      canvasId: "canvas-1",
      branch: "drafts/user-1",
      baseHeadSha: "abc1234567890123456789012345678901234567890",
      files: { [CANVAS_YAML_PATH]: "name: test" },
      updatedAt: Date.now(),
    });

    await clearStaging("canvas-1", "drafts/user-1");

    expect(await getStaging("canvas-1", "drafts/user-1")).toBeNull();
  });

  it("detects whether staging has files", () => {
    expect(hasStagingFiles(null)).toBe(false);
    expect(hasStagingFiles(undefined)).toBe(false);
    expect(
      hasStagingFiles({
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc1234567890123456789012345678901234567890",
        files: {},
        updatedAt: Date.now(),
      }),
    ).toBe(false);
    expect(
      hasStagingFiles({
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc1234567890123456789012345678901234567890",
        files: { [CANVAS_YAML_PATH]: "name: test" },
        updatedAt: Date.now(),
      }),
    ).toBe(true);
  });
});
