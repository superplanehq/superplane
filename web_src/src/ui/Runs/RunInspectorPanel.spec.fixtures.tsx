import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createEvent, fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { vi } from "vitest";
import type {
  ActionsAction,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode,
} from "@/api-client";
import { AccountContext } from "@/contexts/accountContextState";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { RunInspectorPanel } from "./RunInspectorPanel";

export const executions: CanvasesCanvasNodeExecution[] = [
  {
    id: "execution-1",
    nodeId: "action-1",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    resultReason: "RESULT_REASON_ERROR",
    resultMessage: "expression evaluation failed",
    createdAt: "2026-05-01T12:00:01Z",
    updatedAt: "2026-05-01T12:00:02Z",
    outputs: {},
    metadata: {},
    configuration: { retries: 1 },
  },
  {
    id: "execution-2",
    nodeId: "action-2",
    previousExecutionId: "execution-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    createdAt: "2026-05-01T12:00:03Z",
    updatedAt: "2026-05-01T12:00:04Z",
    outputs: { default: [{ data: { ok: true } }] },
    metadata: {},
    configuration: { mode: "create", approvers: [{ type: "anyone" }] },
  },
];

export const runningExecutions: CanvasesCanvasNodeExecution[] = [
  {
    id: "execution-running",
    nodeId: "action-2",
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    createdAt: "2026-05-01T12:00:03Z",
    updatedAt: "2026-05-01T12:00:04Z",
    outputs: {},
    metadata: {},
    configuration: { mode: "create" },
  },
];

export function renderInspector({
  selectedNodeId = null,
  onSelectNode = vi.fn(),
  onClose = vi.fn(),
  run: inspectedRun = run,
  account = null,
}: {
  selectedNodeId?: string | null;
  onSelectNode?: (nodeId: string) => void;
  onClose?: () => void;
  run?: CanvasesCanvasRun;
  account?: {
    id: string;
    name: string;
    email: string;
    avatar_url: string;
    installation_admin: boolean;
    has_password?: boolean;
    roles?: string[];
    groups?: string[];
  } | null;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AccountContext.Provider
        value={{
          account: account ? { ...account, has_password: account.has_password ?? false } : null,
          loading: false,
          setupRequired: false,
        }}
      >
        <ThemeProvider>
          <RunInspectorPanel
            canvasId="canvas-1"
            run={inspectedRun}
            workflowNodes={workflowNodes}
            componentDefinitions={componentDefinitions}
            currentUser={
              account
                ? { id: account.id, email: account.email, roles: account.roles, groups: account.groups }
                : undefined
            }
            selectedNodeId={selectedNodeId}
            onSelectNode={onSelectNode}
            onClose={onClose}
          />
        </ThemeProvider>
      </AccountContext.Provider>
    </QueryClientProvider>,
  );
}

export function renderInteractiveInspector() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  function InteractiveInspector() {
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

    return (
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <RunInspectorPanel
            canvasId="canvas-1"
            run={run}
            workflowNodes={workflowNodes}
            componentDefinitions={componentDefinitions}
            selectedNodeId={selectedNodeId}
            onSelectNode={setSelectedNodeId}
            onClearSelectedNode={() => setSelectedNodeId(null)}
            onClose={vi.fn()}
          />
        </ThemeProvider>
      </QueryClientProvider>
    );
  }

  return render(<InteractiveInspector />);
}

export function firePointerEvent(
  target: Window | Element,
  eventName: "pointerDown" | "pointerMove" | "pointerUp",
  clientX: number,
) {
  const event = createEvent[eventName](target, {});
  Object.defineProperty(event, "pointerId", { value: 1 });
  Object.defineProperty(event, "clientX", { value: clientX });
  fireEvent(target, event);
}

const run: CanvasesCanvasRun = {
  id: "run-1",
  canvasId: "canvas-1",
  state: "STATE_FINISHED",
  result: "RESULT_FAILED",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:05Z",
  rootEvent: {
    id: "event-1",
    nodeId: "trigger-1",
    customName: "Deploy main",
    createdAt: "2026-05-01T12:00:00Z",
    data: { repository: "superplane" },
  },
};

export const runningRun: CanvasesCanvasRun = {
  id: "run-running",
  canvasId: "canvas-1",
  state: "STATE_STARTED",
  result: "RESULT_UNKNOWN",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:05Z",
  rootEvent: {
    id: "event-running",
    nodeId: "trigger-1",
    customName: "Deploy main",
    createdAt: "2026-05-01T12:00:00Z",
    data: { repository: "superplane" },
  },
};

const workflowNodes: SuperplaneComponentsNode[] = [
  {
    id: "trigger-1",
    name: "On Pull Request",
    type: "TYPE_TRIGGER",
    component: "github.onPullRequest",
  },
  {
    id: "action-1",
    name: "Add Grade Label",
    type: "TYPE_ACTION",
    component: "github.addLabel",
  },
  {
    id: "action-2",
    name: "Save Assessment",
    type: "TYPE_ACTION",
    component: "upsertMemory",
  },
  {
    id: "approval-1",
    name: "Await Approval",
    type: "TYPE_ACTION",
    component: "approval",
  },
];

const componentDefinitions: ActionsAction[] = [
  {
    name: "upsertMemory",
    configuration: [
      {
        name: "mode",
        label: "Mode",
        type: "select",
        typeOptions: {
          select: {
            options: [{ label: "Create", value: "create" }],
          },
        },
      },
      {
        name: "approvers",
        label: "Approvers",
        type: "list",
        typeOptions: {
          list: {
            itemLabel: "Approver",
            itemDefinition: {
              type: "object",
              schema: [
                {
                  name: "type",
                  label: "Request approval from",
                  type: "select",
                  typeOptions: {
                    select: {
                      options: [{ label: "Any one", value: "anyone" }],
                    },
                  },
                },
              ],
            },
          },
        },
      },
    ],
  },
];
