export const CANVAS_YAML_PATH = "canvas.yaml";
export const CONSOLE_YAML_PATH = "console.yaml";

export type CanvasStagingRecord = {
  canvasId: string;
  branch: string;
  baseHeadSha: string;
  files: Record<string, string>;
  deletedPaths?: string[];
  updatedAt: number;
};
