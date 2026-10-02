import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { useDefaultRefinementPrompt } from "./useDefaultRefinementPrompt";

const { materializeDefaults } = vi.hoisted(() => ({ materializeDefaults: vi.fn() }));

vi.mock("@/api-client", () => ({
  factoriesMaterializeFactoryAutomationDefaults: materializeDefaults,
}));

const PROMPT = "Plan the task.";

function renderPromptHook() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return renderHook(
    () => useDefaultRefinementPrompt({ organizationId: "org-1", factoryId: "factory-1", automationId: "canvas-1" }),
    { wrapper },
  );
}

function yamlWithPrompt(prompt: string): string {
  return `apiVersion: v1
kind: Canvas
metadata:
  name: Backlog
spec:
  edges: []
  nodes:
    - id: refine-task
      name: Refine Task
      type: TYPE_ACTION
      component: runnerClaudeCode
      configuration:
        steps:
          - name: Refine Task
            type: prompt
            prompt: ${JSON.stringify(prompt)}
`;
}

function yamlWithoutPrompt(): string {
  return `apiVersion: v1
kind: Canvas
metadata:
  name: Backlog
spec:
  edges: []
  nodes:
    - id: refine-task
      name: Refine Task
      type: TYPE_ACTION
      component: runnerClaudeCode
      configuration:
        steps:
          - name: Clone repository
            type: bash
            command: git clone
`;
}

describe("useDefaultRefinementPrompt", () => {
  beforeEach(() => {
    materializeDefaults.mockReset();
  });

  it("reports a load failure when defaults do not include the refinement prompt", async () => {
    materializeDefaults.mockResolvedValue({ data: { canvasYaml: yamlWithoutPrompt() } });

    const { result } = renderPromptHook();

    await waitFor(() => expect(result.current.failed).toBe(true));
    expect(result.current.prompt).toBeUndefined();
  });

  it("returns the refinement prompt when defaults include it", async () => {
    materializeDefaults.mockResolvedValue({ data: { canvasYaml: yamlWithPrompt(PROMPT) } });

    const { result } = renderPromptHook();

    await waitFor(() => expect(result.current.prompt).toBe(PROMPT));
    expect(result.current.failed).toBe(false);
  });

  it("retries a missing prompt load", async () => {
    materializeDefaults
      .mockResolvedValueOnce({ data: { canvasYaml: yamlWithoutPrompt() } })
      .mockResolvedValueOnce({ data: { canvasYaml: yamlWithPrompt(PROMPT) } });

    const { result } = renderPromptHook();

    await waitFor(() => expect(result.current.failed).toBe(true));
    result.current.retry();
    await waitFor(() => expect(result.current.prompt).toBe(PROMPT));
    expect(result.current.failed).toBe(false);
    expect(materializeDefaults).toHaveBeenCalledTimes(2);
  });
});
