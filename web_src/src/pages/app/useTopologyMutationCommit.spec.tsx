import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, SuperplaneComponentsNode } from "@/api-client";
import { DefaultLayoutEngine } from "@/lib/layout";
import { generateUniqueNodeName } from "./utils";
import { appendWorkflowFragment, useTopologyMutationCommit } from "./useTopologyMutationCommit";

function workflowWith(nodes: SuperplaneComponentsNode[] = []): CanvasesCanvas {
  return { metadata: { factoryId: "factory" }, spec: { nodes, edges: [] } };
}

describe("useTopologyMutationCommit", () => {
  it("serializes factory mutations against the latest laid-out workflow", async () => {
    let releaseFirstLayout: ((workflow: CanvasesCanvas) => void) | undefined;
    const firstLayout = new Promise<CanvasesCanvas>((resolve) => {
      releaseFirstLayout = resolve;
    });
    const layoutSpy = vi
      .spyOn(DefaultLayoutEngine, "apply")
      .mockImplementationOnce(() => firstLayout)
      .mockImplementation(async (workflow) => workflow);
    let currentWorkflow = workflowWith();
    const saveWorkflow = vi.fn(async () => undefined);
    const { result } = renderHook(() =>
      useTopologyMutationCommit({
        factoryAutoLayout: true,
        autoLayoutOnUpdate: false,
        components: [],
        getCurrentWorkflow: () => currentWorkflow,
        applyLocalWorkflow: (workflow) => {
          currentWorkflow = workflow;
        },
        saveWorkflow,
        saveSessionRef: { current: 0 },
        readOnly: false,
      }),
    );
    const addNamedNode = (id: string) => (workflow: CanvasesCanvas) => {
      const names = (workflow.spec?.nodes || []).map((node) => node.name || "").filter(Boolean);
      const node: SuperplaneComponentsNode = {
        id,
        name: generateUniqueNodeName("filter", names),
        type: "TYPE_ACTION",
        component: "filter",
      };
      return {
        workflow: appendWorkflowFragment(workflow, [node]),
        options: { addedNodeId: id },
      };
    };
    const firstNode: SuperplaneComponentsNode = {
      id: "first",
      name: "filter",
      type: "TYPE_ACTION",
      component: "filter",
    };

    let firstCommit: Promise<CanvasesCanvas | null>;
    let secondCommit: Promise<CanvasesCanvas | null>;
    act(() => {
      firstCommit = result.current(addNamedNode("first"));
      secondCommit = result.current(addNamedNode("second"));
    });

    await act(async () => Promise.resolve());
    expect(layoutSpy).toHaveBeenCalledTimes(1);
    releaseFirstLayout?.(workflowWith([firstNode]));
    await act(async () => {
      await Promise.all([firstCommit, secondCommit]);
    });

    expect(currentWorkflow.spec?.nodes?.map((node) => node.id)).toEqual(["first", "second"]);
    expect(currentWorkflow.spec?.nodes?.map((node) => node.name)).toEqual(["filter", "filter 2"]);
    expect(saveWorkflow).toHaveBeenCalledTimes(2);
    layoutSpy.mockRestore();
  });

  it("drops queued layout results after the edit session changes", async () => {
    let releaseLayout: ((workflow: CanvasesCanvas) => void) | undefined;
    const layoutResult = new Promise<CanvasesCanvas>((resolve) => {
      releaseLayout = resolve;
    });
    const layoutSpy = vi.spyOn(DefaultLayoutEngine, "apply").mockImplementationOnce(() => layoutResult);
    const applyLocalWorkflow = vi.fn();
    const saveWorkflow = vi.fn(async () => undefined);
    const saveSessionRef = { current: 0 };
    const { result } = renderHook(() =>
      useTopologyMutationCommit({
        factoryAutoLayout: true,
        autoLayoutOnUpdate: false,
        components: [],
        getCurrentWorkflow: () => workflowWith(),
        applyLocalWorkflow,
        saveWorkflow,
        saveSessionRef,
        readOnly: false,
      }),
    );

    const commit = result.current((workflow) => appendWorkflowFragment(workflow, []));
    await act(async () => Promise.resolve());
    saveSessionRef.current += 1;
    releaseLayout?.(workflowWith());
    await act(async () => {
      await commit;
    });

    expect(applyLocalWorkflow).not.toHaveBeenCalled();
    expect(saveWorkflow).not.toHaveBeenCalled();
    layoutSpy.mockRestore();
  });
});
