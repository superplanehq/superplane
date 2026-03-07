import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { WorkflowMarkdownPreview } from "./WorkflowMarkdownPreview";
import { ComponentsNode } from "@/api-client";

type ControlBlock = ControlMarkdownBlock | ControlTableBlock | ControlButtonBlock;
type ControlButtonVariant = "default" | "secondary" | "destructive" | "outline";

interface ControlMarkdownBlock {
  id: string;
  type: "markdown";
  content: string;
}

interface ControlTableBlock {
  id: string;
  type: "table";
  title?: string;
  source:
    | {
        type: "memory";
        namespace: string;
      }
    | {
        type: "static";
        rows: Array<Record<string, unknown>>;
      };
  columns?: ControlTableColumn[];
  actions?: ControlTableAction[];
}

interface ControlTableColumnColorRule {
  when: string;
  color: string;
}

interface ControlTableColumn {
  key: string;
  label?: string;
  color?: string;
  colorRules?: ControlTableColumnColorRule[];
}

interface ControlTableAction {
  id: string;
  label: string;
  nodeId: string;
  channel?: string;
  payload?: unknown;
  confirm?: string;
  variant?: ControlButtonVariant;
}

interface ControlButtonBlock {
  id: string;
  type: "button";
  label: string;
  nodeId: string;
  channel?: string;
  payload?: unknown;
  confirm?: string;
  variant?: ControlButtonVariant;
}

interface ControlConfig {
  blocks: ControlBlock[];
}

interface RunButtonRequest {
  nodeId: string;
  channel: string;
  payload: unknown;
}

interface CanvasControlViewProps {
  memoryEntries: CanvasMemoryEntry[];
  workflowNodes: ComponentsNode[];
  controlConfig?: Record<string, unknown>;
  canRunButtons: boolean;
  runDisabledTooltip?: string;
  onRunButton: (request: RunButtonRequest) => Promise<void>;
}

function findStartTriggerId(workflowNodes: ComponentsNode[], matcher: (name: string) => boolean): string | null {
  for (const node of workflowNodes) {
    const isStartTrigger = node.type === "TYPE_TRIGGER" && node.trigger?.name === "start";
    if (!isStartTrigger) {
      continue;
    }
    const nodeName = (node.name || "").trim().toLowerCase();
    if (node.id && matcher(nodeName)) {
      return node.id;
    }
  }
  return null;
}

function buildMachineControlPreset(workflowNodes: ComponentsNode[]): ControlConfig {
  const spinUpStartId = findStartTriggerId(workflowNodes, (name) => name.includes("spin"));
  const shutdownStartId = findStartTriggerId(workflowNodes, (name) => name.includes("shut"));

  const tableActions: ControlTableAction[] = [];
  if (shutdownStartId) {
    tableActions.push({
      id: "table-shutdown",
      label: "Shutdown",
      nodeId: shutdownStartId,
      channel: "default",
      payload: {
        action: "shutdown",
        machineId: "{{row.machineId}}",
        projectId: "{{row.projectId}}",
        zone: "{{row.zone}}",
      },
      confirm: "Shut down machine {{row.machineId}}?",
      variant: "destructive",
    });
  }

  const blocks: ControlBlock[] = [
    {
      id: "machines-intro",
      type: "markdown",
      content: [
        "# Ephemeral Machines",
        "",
        "Provision, operate, and retire temporary machines for short-lived tasks.",
        "",
        "Use this app to manage machine lifecycle for testing, debugging, and one-off workloads.",
      ].join("\n"),
    },
    {
      id: "machines-table",
      type: "table",
      title: "Ephemeral Machines",
      source: {
        type: "memory",
        namespace: "machines",
      },
      columns: [
        { key: "machineId", label: "Machine ID" },
        {
          key: "status",
          label: "Status",
          colorRules: [
            { when: 'row.status == "ONLINE"', color: "#16a34a" },
            { when: 'row.status == "BOOTING_UP"', color: "#2563eb" },
          ],
        },
        { key: "zone", label: "Zone" },
        { key: "machineType", label: "Type" },
      ],
      actions: tableActions,
    },
  ];

  if (spinUpStartId) {
    blocks.push({
      id: "start-new-machine",
      type: "button",
      label: "Start new machine",
      nodeId: spinUpStartId,
      channel: "default",
      variant: "default",
      payload: {
        action: "spin_up",
        machineId: "ephemeral-{{runTs}}",
        projectId: "my-project",
        zone: "us-central1-a",
        machineType: "e2-medium",
      },
    });
  }

  return { blocks };
}

function isControlConfig(value: unknown): value is ControlConfig {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return false;
  }
  const maybeBlocks = (value as { blocks?: unknown }).blocks;
  return Array.isArray(maybeBlocks);
}

function formatJson(value: unknown): string {
  try {
    const serialized = JSON.stringify(value, null, 2);
    return serialized ?? String(value);
  } catch {
    return String(value);
  }
}

function formatTableCellValue(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  return formatJson(value);
}

function getByPath(source: unknown, path: string): unknown {
  if (!path) {
    return undefined;
  }
  const segments = path.split(".").filter(Boolean);
  let current: unknown = source;
  for (const segment of segments) {
    if (Array.isArray(current)) {
      const index = Number(segment);
      if (Number.isNaN(index)) {
        return undefined;
      }
      current = current[index];
      continue;
    }
    if (typeof current !== "object" || current === null) {
      return undefined;
    }
    current = (current as Record<string, unknown>)[segment];
  }
  return current;
}

function resolveTemplateString(template: string, context: Record<string, unknown>): string {
  return template.replace(/\{\{\s*([^{}]+)\s*\}\}/g, (_match, key) => {
    const value = getByPath(context, String(key).trim());
    if (value === undefined || value === null) {
      return "";
    }
    if (typeof value === "string") {
      return value;
    }
    return formatJson(value);
  });
}

function resolveTemplate(value: unknown, context: Record<string, unknown>): unknown {
  if (typeof value === "string") {
    return resolveTemplateString(value, context);
  }
  if (Array.isArray(value)) {
    return value.map((item) => resolveTemplate(item, context));
  }
  if (typeof value === "object" && value !== null) {
    const resolvedEntries = Object.entries(value).map(
      ([key, nested]) => [key, resolveTemplate(nested, context)] as const,
    );
    return Object.fromEntries(resolvedEntries);
  }
  return value;
}

function parseExpressionValue(token: string, context: Record<string, unknown>): unknown {
  const trimmed = token.trim();
  if (!trimmed) {
    return "";
  }
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return trimmed.slice(1, -1);
  }
  if (trimmed === "true") {
    return true;
  }
  if (trimmed === "false") {
    return false;
  }
  if (trimmed === "null") {
    return null;
  }
  if (trimmed === "undefined") {
    return undefined;
  }

  const numeric = Number(trimmed);
  if (!Number.isNaN(numeric) && trimmed !== "") {
    return numeric;
  }

  if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
    try {
      return JSON.parse(trimmed);
    } catch {
      return trimmed;
    }
  }

  return getByPath(context, trimmed);
}

function evaluateExpressionClause(expression: string, context: Record<string, unknown>): boolean {
  const trimmed = expression.trim();
  if (!trimmed) {
    return false;
  }

  const comparatorMatch = trimmed.match(/^(.*?)\s*(==|!=|>=|<=|>|<|in)\s*(.*?)$/);
  if (!comparatorMatch) {
    return Boolean(parseExpressionValue(trimmed, context));
  }

  const leftValue = parseExpressionValue(comparatorMatch[1] || "", context);
  const operator = comparatorMatch[2] || "";
  const rightValue = parseExpressionValue(comparatorMatch[3] || "", context);

  switch (operator) {
    case "==":
      return leftValue === rightValue;
    case "!=":
      return leftValue !== rightValue;
    case ">":
      return Number(leftValue) > Number(rightValue);
    case "<":
      return Number(leftValue) < Number(rightValue);
    case ">=":
      return Number(leftValue) >= Number(rightValue);
    case "<=":
      return Number(leftValue) <= Number(rightValue);
    case "in":
      return Array.isArray(rightValue) ? rightValue.includes(leftValue) : false;
    default:
      return false;
  }
}

function evaluateExpression(expression: string, context: Record<string, unknown>): boolean {
  const orGroups = expression
    .split("||")
    .map((part) => part.trim())
    .filter(Boolean);

  if (orGroups.length === 0) {
    return false;
  }

  return orGroups.some((group) => {
    const andClauses = group
      .split("&&")
      .map((part) => part.trim())
      .filter(Boolean);
    return andClauses.every((clause) => evaluateExpressionClause(clause, context));
  });
}

function resolveTableCellColor(
  column: ControlTableColumn,
  expressionContext: Record<string, unknown>,
): string | undefined {
  if (column.colorRules?.length) {
    for (const rule of column.colorRules) {
      if (rule.when && evaluateExpression(rule.when, expressionContext)) {
        return rule.color;
      }
    }
  }
  return column.color;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function collectColumns(rows: Array<Record<string, unknown>>): string[] {
  const uniqueColumns = new Set<string>();
  rows.forEach((row) => {
    Object.keys(row).forEach((column) => uniqueColumns.add(column));
  });
  return Array.from(uniqueColumns);
}

function generateRunId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

function RunningIndicator({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center gap-2">
      <span className="h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
      <span>{label}</span>
    </span>
  );
}

export function CanvasControlView({
  memoryEntries,
  workflowNodes,
  controlConfig,
  canRunButtons,
  runDisabledTooltip,
  onRunButton,
}: CanvasControlViewProps) {
  const yamlDerivedConfig = useMemo(() => buildMachineControlPreset(workflowNodes), [workflowNodes]);
  const config = useMemo(
    () => (isControlConfig(controlConfig) ? controlConfig : yamlDerivedConfig),
    [controlConfig, yamlDerivedConfig],
  );
  const [runError, setRunError] = useState<string | null>(null);
  const [runningActionId, setRunningActionId] = useState<string | null>(null);

  const memoryByNamespace = useMemo(() => {
    const grouped: Record<string, unknown[]> = {};
    memoryEntries.forEach((entry) => {
      const namespace = entry.namespace || "(no namespace)";
      if (!grouped[namespace]) {
        grouped[namespace] = [];
      }
      grouped[namespace].push(entry.values);
    });
    return grouped;
  }, [memoryEntries]);

  const templateContext = useMemo(
    () => ({
      memory: memoryByNamespace,
      memoryCount: memoryEntries.length,
      nowIso: new Date().toISOString(),
    }),
    [memoryByNamespace, memoryEntries.length],
  );

  const runConfiguredAction = async (
    action: {
      nodeId: string;
      channel?: string;
      payload?: unknown;
      confirm?: string;
    },
    actionId: string,
    actionContext: Record<string, unknown>,
  ) => {
    if (!canRunButtons) {
      return;
    }

    const runtimeContext = {
      ...actionContext,
      runId: generateRunId(),
      runTs: Date.now(),
      nowIso: new Date().toISOString(),
    };

    const resolvedConfirm = action.confirm ? resolveTemplateString(action.confirm, runtimeContext) : undefined;
    if (resolvedConfirm) {
      const confirmed = window.confirm(resolvedConfirm);
      if (!confirmed) {
        return;
      }
    }

    setRunError(null);
    setRunningActionId(actionId);
    try {
      const resolvedPayload = resolveTemplate(action.payload ?? {}, runtimeContext);
      const resolvedNodeId = resolveTemplateString(action.nodeId, runtimeContext).trim();
      const resolvedChannel = resolveTemplateString(action.channel || "default", runtimeContext).trim() || "default";
      if (!resolvedNodeId) {
        setRunError("Action is missing a valid nodeId.");
        return;
      }
      await onRunButton({
        nodeId: resolvedNodeId,
        channel: resolvedChannel,
        payload: resolvedPayload,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to run button action.";
      setRunError(message);
    } finally {
      setRunningActionId(null);
    }
  };

  const handleRunBlock = async (block: ControlButtonBlock) => {
    await runConfiguredAction(block, block.id, templateContext);
  };

  const renderTableBlock = (block: ControlTableBlock) => {
    const rows: Array<Record<string, unknown>> =
      block.source.type === "memory"
        ? (memoryByNamespace[block.source.namespace] || []).filter(isRecord)
        : block.source.rows.filter(isRecord);

    if (rows.length === 0) {
      return (
        <div className="rounded-md border border-dashed border-slate-200 p-3 text-sm text-slate-500">
          No rows for this table.
        </div>
      );
    }

    const columns: ControlTableColumn[] =
      block.columns && block.columns.length > 0
        ? block.columns.map((column) => ({
            ...column,
            label: column.label || column.key,
          }))
        : collectColumns(rows).map((column) => ({ key: column, label: column }));

    return (
      <div className="overflow-x-auto rounded-md border border-slate-200">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              {columns.map((column) => (
                <th key={column.key} className="px-3 py-2 text-left text-xs font-semibold uppercase text-slate-600">
                  {column.label}
                </th>
              ))}
              {block.actions && block.actions.length > 0 ? (
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase text-slate-600">Actions</th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rowIndex) => (
              <tr key={`${block.id}-${rowIndex}`} className="border-b border-slate-100 last:border-b-0">
                {columns.map((column) => {
                  const expressionContext = {
                    ...templateContext,
                    row,
                    rowIndex,
                    value: row[column.key],
                    columnKey: column.key,
                  };
                  const cellColor = resolveTableCellColor(column, expressionContext);
                  return (
                    <td
                      key={`${block.id}-${rowIndex}-${column.key}`}
                      className="px-3 py-2 align-middle font-mono text-xs text-slate-700"
                      style={cellColor ? { color: cellColor } : undefined}
                    >
                      {formatTableCellValue(row[column.key])}
                    </td>
                  );
                })}
                {block.actions && block.actions.length > 0 ? (
                  <td className="px-3 py-2 align-middle">
                    <div className="flex flex-wrap items-center gap-2">
                      {block.actions.map((action) => {
                        const actionKey = `${block.id}:${rowIndex}:${action.id}`;
                        const rowContext = {
                          ...templateContext,
                          row,
                          rowIndex,
                          table: {
                            id: block.id,
                            title: block.title || "",
                            namespace: block.source.type === "memory" ? block.source.namespace : "",
                          },
                        };
                        return (
                          <Button
                            key={actionKey}
                            type="button"
                            size="sm"
                            variant={action.variant || "outline"}
                            className={runningActionId === actionKey ? "!opacity-100" : undefined}
                            disabled={!canRunButtons || runningActionId === actionKey}
                            onClick={() => {
                              void runConfiguredAction(action, actionKey, rowContext);
                            }}
                          >
                            {runningActionId === actionKey ? <RunningIndicator label={action.label} /> : action.label}
                          </Button>
                        );
                      })}
                    </div>
                    {!canRunButtons ? (
                      <div className="mt-2 text-xs text-slate-500">
                        {runDisabledTooltip || "Buttons are disabled because this canvas cannot run right now."}
                      </div>
                    ) : null}
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  return (
    <div className="h-full overflow-hidden p-10">
      <section className="mx-auto min-h-0 max-w-[900px] overflow-auto rounded-2xl border border-slate-200 bg-white p-10">
        {runError ? <div className="mb-8 text-xs text-red-600">{runError}</div> : null}

        <div className="space-y-10">
          {config.blocks.map((block) => {
            if (block.type === "markdown") {
              const resolvedContent = resolveTemplateString(block.content, templateContext);
              return (
                <div key={block.id}>
                  <WorkflowMarkdownPreview
                    content={resolvedContent}
                    className="[&_h1]:!mt-0 [&_h1]:!mb-4 [&_h1]:!border-b [&_h1]:!border-slate-200 [&_h1]:!pb-2 [&_h1]:!text-[2em] [&_h1]:!font-semibold [&_h1]:!leading-[1.25] [&_h2]:!mt-6 [&_h2]:!mb-3 [&_h2]:!border-b [&_h2]:!border-slate-200 [&_h2]:!pb-1.5 [&_h2]:!text-2xl [&_h2]:!font-semibold [&_h2]:!leading-[1.25] [&_h3]:!mt-6 [&_h3]:!mb-2 [&_h3]:!text-xl [&_h3]:!font-semibold [&_p]:!my-4 [&_p]:!leading-[1.6] [&_ul]:!my-4 [&_ul]:!pl-8 [&_ol]:!my-4 [&_ol]:!pl-8 [&_li]:!my-1 [&_blockquote]:!my-4 [&_blockquote]:!border-l-4 [&_blockquote]:!border-slate-300 [&_blockquote]:!pl-4 [&_code]:!rounded-md [&_code]:!bg-slate-100 [&_code]:!px-1.5 [&_code]:!py-0.5 [&_code]:!text-[85%] [&_pre]:!my-4 [&_pre]:!rounded-md [&_pre]:!border [&_pre]:!border-slate-200 [&_pre]:!bg-slate-50 [&_pre]:!p-4 [&_pre_code]:!bg-transparent [&_pre_code]:!p-0"
                  />
                </div>
              );
            }

            if (block.type === "table") {
              return (
                <div key={block.id} className="space-y-2">
                  {block.title ? <h3 className="text-sm font-semibold text-slate-900">{block.title}</h3> : null}
                  {renderTableBlock(block)}
                </div>
              );
            }

            return (
              <div key={block.id}>
                <Button
                  type="button"
                  variant={block.variant || "default"}
                  className={runningActionId === block.id ? "!opacity-100" : undefined}
                  onClick={() => {
                    void handleRunBlock(block);
                  }}
                  disabled={!canRunButtons || runningActionId === block.id}
                >
                  {runningActionId === block.id ? <RunningIndicator label={block.label} /> : block.label}
                </Button>
                {!canRunButtons ? (
                  <div className="mt-2 text-xs text-slate-500">
                    {runDisabledTooltip || "Buttons are disabled because this canvas cannot run right now."}
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      </section>
    </div>
  );
}
