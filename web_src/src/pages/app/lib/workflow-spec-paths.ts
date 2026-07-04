export const CANVAS_YAML_PATH = "canvas.yaml";
export const CONSOLE_YAML_PATH = "console.yaml";

export const WORKFLOW_SPEC_PATHS = new Set([CANVAS_YAML_PATH, CONSOLE_YAML_PATH]);

export function isWorkflowSpecPath(path: string): boolean {
  return WORKFLOW_SPEC_PATHS.has(path);
}
