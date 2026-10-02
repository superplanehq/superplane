import type { AgentStep } from "./automationsViewModel";

export function bashStepCanOpen(step: AgentStep): boolean {
  if (step.type !== "bash") {
    return false;
  }
  if (step.status === "failed") {
    return true;
  }
  const { script, stdout } = bashStepCommand(step);
  if (stdout.trim()) {
    return true;
  }
  return Boolean(script.trim()) && script.trim() !== step.title.trim();
}

export function bashStepCommand(step: AgentStep): { script: string; stdout: string } {
  if (step.commandScript?.trim()) {
    return { script: step.commandScript, stdout: step.commandStdout ?? "" };
  }
  if (step.output?.trim()) {
    return { script: step.output, stdout: "" };
  }
  return { script: "", stdout: "" };
}

export function bashTitleIsCommand(step: AgentStep, script: string): boolean {
  return !script.trim() || script.trim() === step.title.trim();
}
