import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH, type CanvasStagingRecord } from "@/lib/canvas-staging";

export function resolveStagedBranchYaml(
  staging: CanvasStagingRecord,
  gitCanvasYaml: string,
  gitConsoleYaml: string,
): { canvasYaml: string; consoleYaml: string } {
  return {
    canvasYaml: staging.files[CANVAS_YAML_PATH] ?? gitCanvasYaml,
    consoleYaml: staging.files[CONSOLE_YAML_PATH] ?? gitConsoleYaml,
  };
}
