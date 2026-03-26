import { describe, expect, it, vi } from "vitest";
import { createSafeComponentMapper, createSafeTriggerRenderer } from "./safeMappers";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  SubtitleContext,
  TriggerEventContext,
  TriggerRenderer,
  TriggerRendererContext,
} from "./types";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { TriggerProps } from "@/ui/trigger";

function makeComponentBaseContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: { id: "n1", name: "Test Node", componentName: "test", isCollapsed: false },
    componentDefinition: { name: "test", label: "Test", description: "", icon: "zap", color: "blue" },
    lastExecutions: [],
    ...overrides,
  };
}

function makeSubtitleContext(overrides?: Partial<SubtitleContext>): SubtitleContext {
  return {
    node: { id: "n1", name: "Test Node", componentName: "test", isCollapsed: false },
    execution: {
      id: "e1",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
    },
    ...overrides,
  };
}

function makeExecutionDetailsContext(overrides?: Partial<ExecutionDetailsContext>): ExecutionDetailsContext {
  return {
    nodes: [],
    node: { id: "n1", name: "Test Node", componentName: "test", isCollapsed: false },
    execution: {
      id: "e1",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
    },
    ...overrides,
  };
}

function makeTriggerRendererContext(overrides?: Partial<TriggerRendererContext>): TriggerRendererContext {
  return {
    node: { id: "n1", name: "Test Trigger", componentName: "test", isCollapsed: false },
    definition: { name: "test", label: "Test", description: "", icon: "bolt", color: "blue" },
    lastEvent: { id: "ev1", createdAt: new Date().toISOString(), data: {}, nodeId: "n1", type: "test" },
    ...overrides,
  };
}

function makeTriggerEventContext(overrides?: Partial<TriggerEventContext>): TriggerEventContext {
  return {
    event: { id: "ev1", createdAt: new Date().toISOString(), data: {}, nodeId: "n1", type: "test" },
    ...overrides,
  };
}

describe("createSafeComponentMapper", () => {
  it("delegates to the underlying mapper when no error occurs", () => {
    const expected: ComponentBaseProps = {
      iconSlug: "zap",
      collapsed: false,
      title: "Working",
      includeEmptyState: false,
    };
    const underlying: ComponentBaseMapper = {
      props: () => expected,
      subtitle: () => "Sub",
      getExecutionDetails: () => ({ Key: "Value" }),
    };

    const safe = createSafeComponentMapper(underlying, "test");
    const ctx = makeComponentBaseContext();

    expect(safe.props(ctx)).toBe(expected);
    expect(safe.subtitle(makeSubtitleContext())).toBe("Sub");
    expect(safe.getExecutionDetails(makeExecutionDetailsContext())).toEqual({ Key: "Value" });
  });

  it("recovers from an error in props() and returns fallback props", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: ComponentBaseMapper = {
      props: () => {
        throw new Error("undefined property");
      },
      subtitle: () => "",
      getExecutionDetails: () => ({}),
    };

    const safe = createSafeComponentMapper(underlying, "broken.component");
    const ctx = makeComponentBaseContext();

    const result = safe.props(ctx);
    expect(result.title).toBe("Test Node");
    expect(result.iconSlug).toBe("zap");
    expect(result.includeEmptyState).toBe(true);
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Component mapper "broken.component" threw in props()'),
      expect.any(Error),
    );
    consoleSpy.mockRestore();
  });

  it("recovers from a nil map access in props()", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: ComponentBaseMapper = {
      props: (ctx) => {
        const data = ctx.node.configuration as Record<string, Record<string, Record<string, string>>> | undefined;
        return {
          iconSlug: "zap",
          collapsed: false,
          title: data!.nested.deep.name,
          includeEmptyState: false,
        };
      },
      subtitle: () => "",
      getExecutionDetails: () => ({}),
    };

    const safe = createSafeComponentMapper(underlying, "nil-access");
    const ctx = makeComponentBaseContext({ node: { id: "n1", name: "N", componentName: "test", isCollapsed: false } });

    const result = safe.props(ctx);
    expect(result.title).toBe("N");
    expect(result.includeEmptyState).toBe(true);
    consoleSpy.mockRestore();
  });

  it("recovers from an error in subtitle() and returns empty string", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: ComponentBaseMapper = {
      props: () => ({ iconSlug: "zap", collapsed: false, title: "T", includeEmptyState: false }),
      subtitle: () => {
        throw new TypeError("Cannot read properties of undefined");
      },
      getExecutionDetails: () => ({}),
    };

    const safe = createSafeComponentMapper(underlying, "broken");
    expect(safe.subtitle(makeSubtitleContext())).toBe("");
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getExecutionDetails() and returns empty object", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: ComponentBaseMapper = {
      props: () => ({ iconSlug: "zap", collapsed: false, title: "T", includeEmptyState: false }),
      subtitle: () => "",
      getExecutionDetails: () => {
        throw new Error("null reference");
      },
    };

    const safe = createSafeComponentMapper(underlying, "broken");
    expect(safe.getExecutionDetails(makeExecutionDetailsContext())).toEqual({});
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it("recovers when mapper accesses undefined property on context", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: ComponentBaseMapper = {
      props: (ctx) => {
        const outputs = (ctx.lastExecutions[0].outputs as Record<string, unknown[]>)["missing"];
        return {
          iconSlug: "zap",
          collapsed: false,
          title: (outputs[0] as Record<string, string>).name,
          includeEmptyState: false,
        };
      },
      subtitle: () => "",
      getExecutionDetails: () => ({}),
    };

    const safe = createSafeComponentMapper(underlying, "undefined-prop");
    const ctx = makeComponentBaseContext({
      lastExecutions: [
        {
          id: "e1",
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          resultReason: "RESULT_REASON_OK",
          resultMessage: "",
          metadata: {},
          configuration: {},
          rootEvent: undefined,
          outputs: {},
        },
      ],
    });

    const result = safe.props(ctx);
    expect(result.title).toBe("Test Node");
    consoleSpy.mockRestore();
  });
});

describe("createSafeTriggerRenderer", () => {
  it("delegates to the underlying renderer when no error occurs", () => {
    const expectedProps: TriggerProps = { title: "My Trigger", iconSlug: "bolt", metadata: [] };
    const underlying: TriggerRenderer = {
      getTriggerProps: () => expectedProps,
      getRootEventValues: () => ({ key: "val" }),
      getTitleAndSubtitle: () => ({ title: "T", subtitle: "S" }),
    };

    const safe = createSafeTriggerRenderer(underlying, "test");
    expect(safe.getTriggerProps(makeTriggerRendererContext())).toBe(expectedProps);
    expect(safe.getRootEventValues(makeTriggerEventContext())).toEqual({ key: "val" });
    expect(safe.getTitleAndSubtitle(makeTriggerEventContext())).toEqual({ title: "T", subtitle: "S" });
  });

  it("recovers from an error in getTriggerProps() and returns fallback", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: TriggerRenderer = {
      getTriggerProps: () => {
        throw new Error("boom");
      },
      getRootEventValues: () => ({}),
      getTitleAndSubtitle: () => ({ title: "", subtitle: "" }),
    };

    const safe = createSafeTriggerRenderer(underlying, "broken.trigger");
    const result = safe.getTriggerProps(makeTriggerRendererContext());
    expect(result.title).toBe("Test Trigger");
    expect(result.iconSlug).toBe("bolt");
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Trigger renderer "broken.trigger" threw in getTriggerProps()'),
      expect.any(Error),
    );
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getRootEventValues() and returns empty object", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: TriggerRenderer = {
      getTriggerProps: () => ({ title: "T", iconSlug: "bolt", metadata: [] }),
      getRootEventValues: () => {
        const data = null as unknown as Record<string, string>;
        return { key: data.missing };
      },
      getTitleAndSubtitle: () => ({ title: "", subtitle: "" }),
    };

    const safe = createSafeTriggerRenderer(underlying, "nil-access");
    expect(safe.getRootEventValues(makeTriggerEventContext())).toEqual({});
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getTitleAndSubtitle() and returns safe defaults", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: TriggerRenderer = {
      getTriggerProps: () => ({ title: "T", iconSlug: "bolt", metadata: [] }),
      getRootEventValues: () => ({}),
      getTitleAndSubtitle: () => {
        throw new Error("undefined");
      },
    };

    const safe = createSafeTriggerRenderer(underlying, "broken");
    const result = safe.getTitleAndSubtitle(makeTriggerEventContext());
    expect(result.title).toBe("Event");
    expect(result.subtitle).toBe("");
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getEventState() and returns 'triggered'", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const underlying: TriggerRenderer = {
      getTriggerProps: () => ({ title: "T", iconSlug: "bolt", metadata: [] }),
      getRootEventValues: () => ({}),
      getTitleAndSubtitle: () => ({ title: "", subtitle: "" }),
      getEventState: () => {
        throw new Error("state error");
      },
    };

    const safe = createSafeTriggerRenderer(underlying, "broken");
    expect(safe.getEventState!(makeTriggerEventContext())).toBe("triggered");
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it("does not wrap getEventState when the underlying renderer does not define it", () => {
    const underlying: TriggerRenderer = {
      getTriggerProps: () => ({ title: "T", iconSlug: "bolt", metadata: [] }),
      getRootEventValues: () => ({}),
      getTitleAndSubtitle: () => ({ title: "", subtitle: "" }),
    };

    const safe = createSafeTriggerRenderer(underlying, "no-state");
    expect(safe.getEventState).toBeUndefined();
  });

  it("rest of canvas continues to execute after a mapper panic", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    const brokenMapper: ComponentBaseMapper = {
      props: () => {
        throw new Error("panic in broken mapper");
      },
      subtitle: () => "",
      getExecutionDetails: () => ({}),
    };
    const workingMapper: ComponentBaseMapper = {
      props: () => ({ iconSlug: "check", collapsed: false, title: "Works", includeEmptyState: false }),
      subtitle: () => "ok",
      getExecutionDetails: () => ({ Status: "done" }),
    };

    const safeBroken = createSafeComponentMapper(brokenMapper, "broken");
    const safeWorking = createSafeComponentMapper(workingMapper, "working");

    const ctx = makeComponentBaseContext();

    const brokenResult = safeBroken.props(ctx);
    expect(brokenResult.title).toBe("Test Node");

    const workingResult = safeWorking.props(ctx);
    expect(workingResult.title).toBe("Works");
    expect(workingResult.iconSlug).toBe("check");

    consoleSpy.mockRestore();
  });
});
