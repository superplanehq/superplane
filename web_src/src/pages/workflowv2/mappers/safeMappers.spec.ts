import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import {
  createSafeAdditionalDataBuilder,
  createSafeComponentMapper,
  createSafeCustomFieldRenderer,
  createSafeTriggerRenderer,
  normalizeComponentBaseProps,
  normalizeTriggerProps,
} from "./safeMappers";
import type {
  AdditionalDataBuilderContext,
  ComponentBaseContext,
  ComponentBaseMapper,
  CustomFieldRenderer,
  ExecutionDetailsContext,
  ExecutionInfo,
  SubtitleContext,
  TriggerEventContext,
  TriggerRenderer,
  TriggerRendererContext,
} from "./types";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { TriggerProps } from "@/ui/trigger";

const DEFAULT_NODE = { id: "n1", name: "Test Node", componentName: "test", isCollapsed: false };
const DEFAULT_DEFINITION = { name: "test", label: "Test", description: "", icon: "zap", color: "blue" };
const DEFAULT_TRIGGER_NODE = { id: "n1", name: "Test Trigger", componentName: "test", isCollapsed: false };
const DEFAULT_TRIGGER_DEF = { name: "test", label: "Test", description: "", icon: "bolt", color: "blue" };

function makeExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
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
    ...overrides,
  };
}

function makeComponentBaseContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return { nodes: [], node: DEFAULT_NODE, componentDefinition: DEFAULT_DEFINITION, lastExecutions: [], ...overrides };
}

function makeSubtitleContext(): SubtitleContext {
  return { node: DEFAULT_NODE, execution: makeExecution() };
}

function makeExecutionDetailsContext(): ExecutionDetailsContext {
  return { nodes: [], node: DEFAULT_NODE, execution: makeExecution() };
}

function makeTriggerRendererContext(): TriggerRendererContext {
  return {
    node: DEFAULT_TRIGGER_NODE,
    definition: DEFAULT_TRIGGER_DEF,
    lastEvent: { id: "ev1", createdAt: new Date().toISOString(), data: {}, nodeId: "n1", type: "test" },
  };
}

function makeTriggerEventContext(): TriggerEventContext {
  return { event: { id: "ev1", createdAt: new Date().toISOString(), data: {}, nodeId: "n1", type: "test" } };
}

function makeAdditionalDataContext(): AdditionalDataBuilderContext {
  return {
    nodes: [DEFAULT_NODE],
    node: DEFAULT_NODE,
    componentDefinition: DEFAULT_DEFINITION,
    lastExecutions: [],
    canvasId: "canvas-1",
    queryClient: new QueryClient(),
    organizationId: "org-1",
  };
}

function throwingMapper(method: "props" | "subtitle" | "getExecutionDetails"): ComponentBaseMapper {
  const noop: ComponentBaseMapper = {
    props: () => ({ iconSlug: "zap", collapsed: false, title: "T", includeEmptyState: false }),
    subtitle: () => "",
    getExecutionDetails: () => ({}),
  };
  return {
    ...noop,
    [method]: () => {
      throw new Error(`fail in ${method}`);
    },
  };
}

function baseTriggerRenderer(): TriggerRenderer {
  return {
    getTriggerProps: () => ({ title: "T", iconSlug: "bolt", metadata: [] }),
    getRootEventValues: () => ({}),
    getTitleAndSubtitle: () => ({ title: "", subtitle: "" }),
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

    expect(safe.props(makeComponentBaseContext())).toMatchObject(expected);
    expect(safe.subtitle(makeSubtitleContext())).toBe("Sub");
    expect(safe.getExecutionDetails(makeExecutionDetailsContext())).toEqual({ Key: "Value" });
  });

  it("recovers from an error in props() and returns fallback props", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeComponentMapper(throwingMapper("props"), "broken.component");
    const result = safe.props(makeComponentBaseContext());

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
        return { iconSlug: "zap", collapsed: false, title: data!.nested.deep.name, includeEmptyState: false };
      },
      subtitle: () => "",
      getExecutionDetails: () => ({}),
    };
    const safe = createSafeComponentMapper(underlying, "nil-access");
    const ctx = makeComponentBaseContext({ node: { ...DEFAULT_NODE, name: "N" } });

    expect(safe.props(ctx).title).toBe("N");
    expect(safe.props(ctx).includeEmptyState).toBe(true);
    consoleSpy.mockRestore();
  });

  it("recovers from an error in subtitle() and returns empty string", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeComponentMapper(throwingMapper("subtitle"), "broken");
    expect(safe.subtitle(makeSubtitleContext())).toBe("");
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getExecutionDetails() and returns empty object", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeComponentMapper(throwingMapper("getExecutionDetails"), "broken");
    expect(safe.getExecutionDetails(makeExecutionDetailsContext())).toEqual({});
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
    const ctx = makeComponentBaseContext({ lastExecutions: [makeExecution({ outputs: {} })] });

    expect(safe.props(ctx).title).toBe("Test Node");
    consoleSpy.mockRestore();
  });

  it("normalizes malformed mapper props into safe values", () => {
    const result = normalizeComponentBaseProps(
      {
        title: undefined,
        metadata: "bad",
        specs: "bad",
        eventSections: "bad",
        error: { message: "boom" },
        warning: 123,
      } as unknown as ComponentBaseProps,
      makeComponentBaseContext(),
    );

    expect(result.title).toBe("Test Node");
    expect(result.metadata).toBeUndefined();
    expect(result.specs).toBeUndefined();
    expect(result.eventSections).toBeUndefined();
    expect(result.error).toBe("");
    expect(result.warning).toBe("");
    expect(result.includeEmptyState).toBe(true);
    expect(result.emptyStateProps?.title).toBe("Unavailable");
  });
});

describe("createSafeTriggerRenderer", () => {
  it("delegates to the underlying renderer when no error occurs", () => {
    const expectedProps: TriggerProps = { title: "My Trigger", iconSlug: "bolt", metadata: [] };
    const underlying: TriggerRenderer = {
      ...baseTriggerRenderer(),
      getTriggerProps: () => expectedProps,
      getRootEventValues: () => ({ key: "val" }),
      getTitleAndSubtitle: () => ({ title: "T", subtitle: "S" }),
    };
    const safe = createSafeTriggerRenderer(underlying, "test");

    expect(safe.getTriggerProps(makeTriggerRendererContext())).toMatchObject(expectedProps);
    expect(safe.getRootEventValues(makeTriggerEventContext())).toEqual({ key: "val" });
    expect(safe.getTitleAndSubtitle(makeTriggerEventContext())).toEqual({ title: "T", subtitle: "S" });
  });

  it("recovers from an error in getTriggerProps() and returns fallback", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeTriggerRenderer(
      {
        ...baseTriggerRenderer(),
        getTriggerProps: () => {
          throw new Error("boom");
        },
      },
      "broken.trigger",
    );
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
    const safe = createSafeTriggerRenderer(
      {
        ...baseTriggerRenderer(),
        getRootEventValues: () => {
          const d = null as unknown as Record<string, string>;
          return { key: d.x };
        },
      },
      "nil-access",
    );
    expect(safe.getRootEventValues(makeTriggerEventContext())).toEqual({});
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getTitleAndSubtitle() and returns safe defaults", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeTriggerRenderer(
      {
        ...baseTriggerRenderer(),
        getTitleAndSubtitle: () => {
          throw new Error("undefined");
        },
      },
      "broken",
    );
    const result = safe.getTitleAndSubtitle(makeTriggerEventContext());
    expect(result.title).toBe("Event");
    expect(result.subtitle).toBe("");
    consoleSpy.mockRestore();
  });

  it("recovers from an error in getEventState() and returns 'triggered'", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeTriggerRenderer(
      {
        ...baseTriggerRenderer(),
        getEventState: () => {
          throw new Error("state error");
        },
      },
      "broken",
    );
    expect(safe.getEventState!(makeTriggerEventContext())).toBe("triggered");
    consoleSpy.mockRestore();
  });

  it("does not wrap getEventState when the underlying renderer does not define it", () => {
    const safe = createSafeTriggerRenderer(baseTriggerRenderer(), "no-state");
    expect(safe.getEventState).toBeUndefined();
  });

  it("normalizes malformed trigger props into safe values", () => {
    const result = normalizeTriggerProps(
      {
        title: undefined,
        iconSlug: undefined,
        metadata: "bad",
        error: 1,
      } as unknown as TriggerProps,
      makeTriggerRendererContext(),
    );

    expect(result.title).toBe("Test Trigger");
    expect(result.iconSlug).toBe("bolt");
    expect(result.metadata).toEqual([]);
    expect(result.error).toBe("");
  });
});

describe("safe mapper canvas resilience", () => {
  it("rest of canvas continues to execute after a mapper panic", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safeBroken = createSafeComponentMapper(throwingMapper("props"), "broken");
    const safeWorking = createSafeComponentMapper(
      {
        props: () => ({ iconSlug: "check", collapsed: false, title: "Works", includeEmptyState: false }),
        subtitle: () => "ok",
        getExecutionDetails: () => ({ Status: "done" }),
      },
      "working",
    );
    const ctx = makeComponentBaseContext();

    expect(safeBroken.props(ctx).title).toBe("Test Node");
    expect(safeWorking.props(ctx).title).toBe("Works");
    expect(safeWorking.props(ctx).iconSlug).toBe("check");
    consoleSpy.mockRestore();
  });
});

describe("additional data and custom field safety", () => {
  it("returns undefined when an additional data builder throws", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeAdditionalDataBuilder(
      {
        buildAdditionalData: () => {
          throw new Error("builder failed");
        },
      },
      "approval",
    );

    expect(safe.buildAdditionalData(makeAdditionalDataContext())).toBeUndefined();
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Additional data builder "approval" threw in buildAdditionalData()'),
      expect.any(Error),
    );
    consoleSpy.mockRestore();
  });

  it("returns null when a custom field renderer throws", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const safe = createSafeCustomFieldRenderer(
      {
        render: () => {
          throw new Error("render failed");
        },
      } satisfies CustomFieldRenderer,
      "github.runWorkflow",
    );

    expect(safe.render(DEFAULT_NODE)).toBeNull();
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Custom field renderer "github.runWorkflow" threw in render()'),
      expect.any(Error),
    );
    consoleSpy.mockRestore();
  });

  it("returns null when a custom field renderer returns a non-renderable object", () => {
    const safe = createSafeCustomFieldRenderer(
      {
        render: () => ({ invalid: true }) as unknown as ReactNode,
      } satisfies CustomFieldRenderer,
      "broken.customField",
    );

    expect(safe.render(DEFAULT_NODE)).toBeNull();
  });
});
