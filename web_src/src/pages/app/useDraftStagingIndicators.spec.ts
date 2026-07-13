import { describe, expect, it } from "vitest";

import { resolveEditingStagingFlags } from "./useDraftStagingIndicators";

describe("resolveEditingStagingFlags", () => {
  const ready = {
    isEditing: true,
    editBootstrapReady: true,
    committedBaselinesReady: true,
    localHasCanvasStaging: false,
    localHasConsoleStaging: false,
  };

  it("uses server file staging before the user edits files in the session", () => {
    expect(
      resolveEditingStagingFlags({
        ...ready,
        filesLocalStagingActive: false,
        localHasFilesStaging: false,
        serverHasFilesStaging: true,
      }),
    ).toMatchObject({ hasFilesStagingChanges: true, hasStagingChanges: true });
  });

  it("uses local file staging while the user is actively editing files", () => {
    expect(
      resolveEditingStagingFlags({
        ...ready,
        filesLocalStagingActive: true,
        localHasFilesStaging: true,
        serverHasFilesStaging: false,
      }),
    ).toMatchObject({ hasFilesStagingChanges: true, hasStagingChanges: true });
  });

  it("ignores stale server staging after the user reverts all local file edits", () => {
    expect(
      resolveEditingStagingFlags({
        ...ready,
        filesLocalStagingActive: true,
        localHasFilesStaging: false,
        serverHasFilesStaging: true,
      }),
    ).toMatchObject({ hasFilesStagingChanges: false, hasStagingChanges: false });
  });

  it("falls back to server file staging after re-entering edit mode with cleared local pending", () => {
    expect(
      resolveEditingStagingFlags({
        ...ready,
        filesLocalStagingActive: false,
        localHasFilesStaging: false,
        serverHasFilesStaging: true,
      }),
    ).toMatchObject({ hasFilesStagingChanges: true, hasStagingChanges: true });
  });
});
