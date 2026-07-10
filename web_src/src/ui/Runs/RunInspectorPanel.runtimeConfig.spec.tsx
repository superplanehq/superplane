import { fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasVersion, ConfigurationField } from "@/api-client";
import { executions, run as baseRun, renderInspector, workflowNodes } from "./RunInspectorPanel.spec.fixtures";

let mockedExecutions = executions;
let mockedRunVersion: CanvasesCanvasVersion | undefined = undefined;

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value, collapsed }: { value: unknown; collapsed?: boolean | number }) => (
    <pre data-testid="json-view" data-collapsed={String(collapsed)}>
      {JSON.stringify(value)}
    </pre>
  ),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: mockedExecutions },
    isLoading: false,
  }),
  useCanvasVersion: () => ({
    data: mockedRunVersion,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
}));

vi.mock("@/pages/app/mappers", () => ({
  getExecutionDetails: () => ({}),
  getState: () => (execution: CanvasesCanvasNodeExecution) =>
    execution.result === "RESULT_FAILED" ? "error" : "success",
  getStateMap: () => ({
    error: { badgeColor: "bg-red-500", label: "error" },
    success: { badgeColor: "bg-emerald-500", label: "success" },
    triggered: { badgeColor: "bg-blue-500", label: "triggered" },
  }),
  getTriggerRenderer: () => ({
    getTitleAndSubtitle: () => ({ title: "Deploy main", subtitle: "" }),
    getRootEventValues: () => ({ Source: "manual" }),
  }),
}));

vi.mock("@/pages/app/utils", () => ({
  buildEventInfo: (event: unknown) => event,
  buildExecutionInfo: (execution: unknown) => execution,
}));

beforeEach(() => {
  mockedExecutions = executions;
  mockedRunVersion = undefined;
});

afterEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("RunInspectorPanel runtime config expression preview", () => {
  it("highlights resolved runtime config values using versioned workflow expressions", () => {
    const resolvedMessage = "Hello, World!\n\n\nasdasfkgdokgf;dkgodkfgopdfkgpfodkgodfk ofdkgodfkgo dkfgop dkfgo";
    mockedRunVersion = versionWithActionConfiguration({ url: "http://{{ root().data.message }}" });
    mockedExecutions = withActionExecutionConfiguration({ url: `http://${resolvedMessage}` });

    renderInspector({
      selectedNodeId: "action-1",
      run: {
        ...baseRun,
        versionId: "version-used-by-run",
        rootEvent: baseRun.rootEvent
          ? { ...baseRun.rootEvent, data: { data: { message: "client-side value should not render" } } }
          : baseRun.rootEvent,
      },
      workflowNodes: workflowNodes.map((node) =>
        node.id === "action-1"
          ? { ...node, configuration: { url: "https://{{ root().data.currentConfigShouldNotBeUsed }}" } }
          : node,
      ),
      componentDefinitions: [actionDefinition([{ name: "url", label: "URL", type: "string" }])],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByText("URL")).toBeInTheDocument();
    expect(screen.queryByText(/client-side value should not render/)).not.toBeInTheDocument();
    expect(screen.queryByText(/currentConfigShouldNotBeUsed/)).not.toBeInTheDocument();
    const resolvedSegment = screen
      .getAllByText((_, element) => {
        const className = typeof element?.className === "string" ? element.className : "";
        return className.includes("text-emerald-700") && element?.textContent?.includes(resolvedMessage) === true;
      })
      .find((element) => element.className.includes("text-emerald-700"));
    expect(resolvedSegment).toBeDefined();
    expect(resolvedSegment!.parentElement?.textContent).toBe(`http://{{ ${resolvedMessage} }}`);

    fireEvent.click(screen.getByRole("button", { name: "Show expression" }));

    expect(screen.getByText("root().data.message")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show applied" })).toBeInTheDocument();
  });

  it("does not use live workflow expressions while a run version is unavailable", () => {
    mockedExecutions = withActionExecutionConfiguration({ url: "http://server-applied-value" });

    renderInspector({
      selectedNodeId: "action-1",
      run: { ...baseRun, versionId: "version-still-loading" },
      workflowNodes: liveWorkflowWithUrlExpression(),
      componentDefinitions: [actionDefinition([{ name: "url", label: "URL", type: "string" }])],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByText("http://server-applied-value")).toBeInTheDocument();
    expect(screen.queryByText(/currentConfigShouldNotBeUsed/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Show expression" })).not.toBeInTheDocument();
  });

  it("does not enable expression templates for empty version workflow nodes", () => {
    mockedRunVersion = {
      metadata: { id: "empty-version" },
      spec: { nodes: [] },
    };
    mockedExecutions = withActionExecutionConfiguration({ url: "http://server-applied-value" });

    renderInspector({
      selectedNodeId: "action-1",
      run: { ...baseRun, versionId: "empty-version" },
      workflowNodes: liveWorkflowWithUrlExpression(),
      componentDefinitions: [actionDefinition([{ name: "url", label: "URL", type: "string" }])],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByText("http://server-applied-value")).toBeInTheDocument();
    expect(screen.queryByText(/currentConfigShouldNotBeUsed/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Show expression" })).not.toBeInTheDocument();
  });

  it("does not use live workflow expressions when a run has no version id", () => {
    mockedExecutions = withActionExecutionConfiguration({ url: "http://server-applied-value" });

    renderInspector({
      selectedNodeId: "action-1",
      workflowNodes: liveWorkflowWithUrlExpression(),
      componentDefinitions: [actionDefinition([{ name: "url", label: "URL", type: "string" }])],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByText("http://server-applied-value")).toBeInTheDocument();
    expect(screen.queryByText(/currentConfigShouldNotBeUsed/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Show expression" })).not.toBeInTheDocument();
  });

  it("shows field-specific runtime config expression errors below the field", () => {
    const expressionError =
      'error resolving field url: expression evaluation failed: cannot fetch ss from <nil> (1:45) | $["start 2"].data.dasdkasdkaospdkoaskdopad.ss | ............................................^';
    mockedExecutions = withActionExecutionConfiguration(
      {
        url: '{{ $["start 2"].data.dasdkasdkaospdkoaskdopad.ss }}',
      },
      expressionError,
    );

    renderInspector({
      selectedNodeId: "action-1",
      componentDefinitions: [actionDefinition([{ name: "url", label: "URL", type: "string" }])],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    const highlightedExpression = screen
      .getAllByText(/start 2.*dasdkasdkaospdkoaskdopad\.ss/)
      .find((element) => element.className.includes("underline"));
    expect(highlightedExpression).toBeDefined();
    expect(highlightedExpression!).toHaveClass("text-red-600");
    const fieldError = screen.getByTestId("runtime-config-expression-error-url");
    expect(fieldError).toHaveTextContent("expression evaluation failed: cannot fetch ss from <nil>");
    expect(fieldError).not.toHaveTextContent("$[");
    expect(fieldError).not.toHaveTextContent(/error resolving field url/i);
  });

  it("shows expression errors for boolean runtime config fields", () => {
    const expressionError =
      "error resolving field enabled: expression evaluation failed: cannot fetch enabled from <nil> (1:14) | $.data.enabled | .............^";
    mockedExecutions = withActionExecutionConfiguration({ enabled: "{{ $.data.enabled }}" }, expressionError);

    renderInspector({
      selectedNodeId: "action-1",
      componentDefinitions: [actionDefinition([{ name: "enabled", label: "Enabled", type: "boolean" }])],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    const highlightedExpression = screen
      .getAllByText(/\.data\.enabled/)
      .find((element) => element.className.includes("underline"));
    expect(highlightedExpression).toBeDefined();
    expect(highlightedExpression!).toHaveClass("text-red-600");
    expect(screen.getByTestId("runtime-config-expression-error-enabled")).toHaveTextContent(
      "expression evaluation failed: cannot fetch enabled from <nil>",
    );
  });

  it("matches runtime config expression errors by expression text when the backend field path differs", () => {
    const expressionError =
      'error resolving field request.url: expression evaluation failed: cannot fetch ss from <nil> (1:45) | $["start 2"].data.dasdkasdkaospdkoaskdopad.ss | ............................................^';
    mockedRunVersion = versionWithActionConfiguration({
      endpoint: '{{ $["start 2"].data.dasdkasdkaospdkoaskdopad.ss }}',
      url: "{{ root().data.url }}",
    });
    mockedExecutions = withActionExecutionConfiguration(
      { endpoint: "", url: "http://server-applied-value" },
      expressionError,
    );

    renderInspector({
      selectedNodeId: "action-1",
      run: { ...baseRun, versionId: "version-used-by-run" },
      componentDefinitions: [
        actionDefinition([
          { name: "url", label: "URL", type: "string" },
          { name: "endpoint", label: "Endpoint", type: "string" },
        ]),
      ],
    });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByTestId("runtime-config-expression-error-endpoint")).toHaveTextContent(
      "expression evaluation failed: cannot fetch ss from <nil>",
    );
    expect(screen.queryByTestId("runtime-config-expression-error-url")).not.toBeInTheDocument();
  });

  it("preserves newlines in fallback runtime config strings", () => {
    mockedExecutions = withActionExecutionConfiguration({ message: "first line\nsecond line" });

    renderInspector({ selectedNodeId: "action-1" });
    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByRole("textbox", { name: "Message" })).toHaveValue("first line\nsecond line");
  });
});

function versionWithActionConfiguration(configuration: Record<string, unknown>): CanvasesCanvasVersion {
  return {
    metadata: { id: "version-used-by-run" },
    spec: {
      nodes: workflowNodes.map((node) => (node.id === "action-1" ? { ...node, configuration } : node)),
    },
  };
}

function withActionExecutionConfiguration(
  configuration: Record<string, unknown>,
  resultMessage = "",
): CanvasesCanvasNodeExecution[] {
  return executions.map((execution) =>
    execution.nodeId === "action-1"
      ? {
          ...execution,
          result: resultMessage ? execution.result : "RESULT_PASSED",
          resultReason: resultMessage ? execution.resultReason : "RESULT_REASON_OK",
          resultMessage,
          configuration,
        }
      : execution,
  );
}

function liveWorkflowWithUrlExpression() {
  return workflowNodes.map((node) =>
    node.id === "action-1"
      ? { ...node, configuration: { url: "https://{{ root().data.currentConfigShouldNotBeUsed }}" } }
      : node,
  );
}

function actionDefinition(configuration: ConfigurationField[]) {
  return {
    name: "github.addLabel",
    configuration,
  };
}
