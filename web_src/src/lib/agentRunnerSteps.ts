/** Titles of configured bash and prompt steps, in execution order. */
export function agentRunnerStepTitles(configuration: unknown): string[] {
  if (!configuration || typeof configuration !== "object") {
    return [];
  }
  const steps = (configuration as Record<string, unknown>).steps;
  if (!Array.isArray(steps)) {
    return [];
  }

  return steps.flatMap((step) => {
    if (!step || typeof step !== "object") {
      return [];
    }
    const name = (step as Record<string, unknown>).name;
    return typeof name === "string" && name.trim() ? [name.trim()] : [];
  });
}

export const AGENT_HARNESS_COMPONENTS = new Set<string>([
  "runnerSuperPlane",
  "runnerClaudeCode",
  "runnerCodex",
  "runnerOpenRouter",
]);

export function isAgentHarnessComponent(component: string | undefined): boolean {
  return Boolean(component && AGENT_HARNESS_COMPONENTS.has(component));
}
