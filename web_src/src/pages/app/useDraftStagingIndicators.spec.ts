import { describe, expect, it } from "vitest";

import { resolveEditingStagingFlags } from "./useDraftStagingIndicators";

describe("resolveEditingStagingFlags", () => {
  it("uses server staging flags before committed baselines are ready", () => {
    expect(
      resolveEditingStagingFlags({
        isEditing: true,
        committedBaselinesReady: false,
        localHasCanvasStaging: false,
        localHasConsoleStaging: false,
        serverHasCanvasStaging: true,
        serverHasConsoleStaging: false,
        serverHasStagingChanges: true,
        filesLocalStagingActive: false,
        localHasFilesStaging: false,
        serverHasFilesStaging: false,
      }),
    ).toMatchObject({
      hasCanvasStagingChanges: true,
      hasStagingChanges: true,
    });
  });

  it("keeps commit controls visible when local diffs lag behind server staging", () => {
    expect(
      resolveEditingStagingFlags({
        isEditing: true,
        committedBaselinesReady: true,
        localHasCanvasStaging: false,
        localHasConsoleStaging: false,
        serverHasCanvasStaging: true,
        serverHasConsoleStaging: false,
        serverHasStagingChanges: true,
        filesLocalStagingActive: false,
        localHasFilesStaging: false,
        serverHasFilesStaging: false,
      }),
    ).toMatchObject({
      hasCanvasStagingChanges: true,
      hasStagingChanges: true,
    });
  });

  it("hides staging controls outside edit mode", () => {
    expect(
      resolveEditingStagingFlags({
        isEditing: false,
        committedBaselinesReady: true,
        localHasCanvasStaging: true,
        localHasConsoleStaging: true,
        serverHasCanvasStaging: true,
        serverHasConsoleStaging: true,
        serverHasStagingChanges: true,
        filesLocalStagingActive: true,
        localHasFilesStaging: true,
        serverHasFilesStaging: true,
      }),
    ).toMatchObject({
      hasStagingChanges: false,
    });
  });
});
