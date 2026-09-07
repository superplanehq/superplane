import { useMemo, useState } from "react";
import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import {
  agentRunCacheReadCount,
  agentRunInputTokenCount,
  agentRunOutputTokenCount,
  agentRunToolCallCount,
  chartPointsForTelemetry,
  type AgentChartMode,
  type AgentChartPoint,
  type AgentRunTelemetry,
  type AgentTurnTool,
} from "@/lib/agentRunTelemetry";
import { cn } from "@/lib/utils";

type AgentRunUsageChartProps = {
  telemetry: AgentRunTelemetry;
  title?: string;
};

const CHART_HEIGHT = 200;
const PAD_LEFT = 40;
const PAD_RIGHT = 12;
const PAD_TOP = 12;
const PAD_BOTTOM = 28;
const BAR_SLOT = 36;
const TOOL_TEXT_COLLAPSE_LIMIT = 80;
const MESSAGE_COLLAPSE_LIMIT = 220;
const INPUT_BAR_FILL = "fill-sky-500 dark:fill-sky-400";
const OUTPUT_BAR_FILL = "fill-amber-500 dark:fill-amber-400";
const INPUT_SWATCH = "bg-sky-500 dark:bg-sky-400";
const OUTPUT_SWATCH = "bg-amber-500 dark:bg-amber-400";

export function AgentRunUsageChart({ telemetry, title }: AgentRunUsageChartProps) {
  const [mode, setMode] = useState<AgentChartMode>("per-turn");
  const [hoveredTurn, setHoveredTurn] = useState<number | null>(null);
  const [pinnedTurn, setPinnedTurn] = useState<number | null>(null);
  const points = useMemo(() => chartPointsForTelemetry(telemetry, mode), [telemetry, mode]);
  const activeTurn = pinnedTurn ?? hoveredTurn;
  const active = points.find((point) => point.turn === activeTurn) ?? null;

  if (telemetry.turns.length === 0) {
    return (
      <section className="min-w-0">
        {title ? <h3 className="mb-2 text-sm font-medium text-foreground">{title}</h3> : null}
        <p className="py-2 text-sm text-muted-foreground">No usage data yet.</p>
      </section>
    );
  }

  return (
    <section className="min-w-0 overflow-hidden">
      {title ? <h3 className="mb-2 text-sm font-medium text-foreground">{title}</h3> : null}
      <p className="mb-3 text-sm text-muted-foreground">{usageSummary(telemetry)}</p>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <TokenKindLegend />
        <ChartToggle
          value={mode}
          options={[
            { id: "per-turn", label: "Per turn" },
            { id: "cumulative", label: "Cumulative" },
          ]}
          onChange={setMode}
        />
      </div>
      <div className="min-w-0" onMouseLeave={() => setHoveredTurn(null)}>
        <UsagePlot
          points={points}
          hoveredTurn={hoveredTurn}
          pinnedTurn={pinnedTurn}
          onHoverTurn={setHoveredTurn}
          onPinTurn={setPinnedTurn}
        />
        <TurnHoverDetails point={active} />
      </div>
    </section>
  );
}

function TokenKindLegend() {
  return (
    <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
      <span className="inline-flex items-center gap-1.5">
        <span className={cn("size-2.5 rounded-sm", INPUT_SWATCH)} />
        Input
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className={cn("size-2.5 rounded-sm", OUTPUT_SWATCH)} />
        Output
      </span>
    </div>
  );
}

function ChartToggle<T extends string>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: Array<{ id: T; label: string }>;
  onChange: (value: T) => void;
}) {
  return (
    <div className="inline-flex rounded-md border border-border p-0.5 text-xs">
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          className={cn(
            "rounded px-2 py-1 font-medium",
            value === option.id ? "bg-foreground text-background" : "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => onChange(option.id)}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

function UsagePlot({
  points,
  hoveredTurn,
  pinnedTurn,
  onHoverTurn,
  onPinTurn,
}: {
  points: AgentChartPoint[];
  hoveredTurn: number | null;
  pinnedTurn: number | null;
  onHoverTurn: (turn: number | null) => void;
  onPinTurn: (turn: number | null) => void;
}) {
  const plotHeight = CHART_HEIGHT - PAD_TOP - PAD_BOTTOM;
  const chartWidth = Math.max(640, PAD_LEFT + PAD_RIGHT + points.length * BAR_SLOT);
  const plotWidth = chartWidth - PAD_LEFT - PAD_RIGHT;
  const maxValue = Math.max(1, ...points.map((point) => point.inputBar + point.outputBar));
  const step = points.length > 1 ? plotWidth / (points.length - 1) : 0;
  const columnX = (index: number) => PAD_LEFT + (points.length === 1 ? plotWidth / 2 : index * step);

  return (
    <div className="overflow-x-auto">
      <svg
        role="img"
        aria-label="Input and output tokens by turn"
        viewBox={`0 0 ${chartWidth} ${CHART_HEIGHT}`}
        className="h-52 w-full min-w-[40rem]"
      >
        <text x={4} y={PAD_TOP + 10} className="fill-current text-[10px] text-muted-foreground" pointerEvents="none">
          {formatCompactTokenValue(maxValue)}
        </text>
        {points.map((point, index) => {
          const x = columnX(index);
          const stack = stackedBarLayout(point, maxValue, plotHeight);
          const hovered = hoveredTurn === point.turn;
          const pinned = pinnedTurn === point.turn;
          return (
            <g key={point.turn} pointerEvents="none">
              {point.inputBar > 0 ? (
                <rect
                  x={x - 10}
                  y={stack.inputTop}
                  width={20}
                  height={Math.max(stack.inputHeight, 1)}
                  className={INPUT_BAR_FILL}
                />
              ) : null}
              {point.outputBar > 0 ? (
                <rect
                  x={x - 10}
                  y={stack.outputTop}
                  width={20}
                  height={Math.max(stack.outputHeight, 1)}
                  className={OUTPUT_BAR_FILL}
                />
              ) : null}
              {hovered || pinned ? (
                <rect
                  x={x - 12}
                  y={stack.top - 2}
                  width={24}
                  height={stack.height + 4}
                  rx={4}
                  className={cn("fill-none", pinned ? "stroke-foreground stroke-2" : "stroke-foreground/60 stroke-1")}
                />
              ) : null}
              <text
                x={x}
                y={CHART_HEIGHT - 8}
                textAnchor="middle"
                className="fill-current text-[10px] text-muted-foreground"
              >
                {point.turn}
              </text>
            </g>
          );
        })}
        {points.map((point, index) => {
          const x = columnX(index);
          const pinned = pinnedTurn === point.turn;
          return (
            <rect
              key={`hit-${point.turn}`}
              x={x - 14}
              y={PAD_TOP}
              width={28}
              height={plotHeight}
              role="button"
              aria-pressed={pinned}
              aria-label={turnAriaLabel(point)}
              className="cursor-pointer fill-transparent"
              pointerEvents="all"
              onMouseEnter={() => onHoverTurn(point.turn)}
              onClick={() => onPinTurn(pinned ? null : point.turn)}
            />
          );
        })}
      </svg>
    </div>
  );
}

function TurnHoverDetails({ point }: { point: AgentChartPoint | null }) {
  return (
    <div
      data-testid="agent-run-turn-details"
      className="mt-3 h-48 min-w-0 overflow-y-auto rounded-md border border-border bg-muted/40 px-3 py-2 text-sm"
    >
      {point ? (
        <>
          <p className="font-medium text-foreground">{turnDetailsTitle(point)}</p>
          {point.cache_read_tokens > 0 ? (
            <p className="mt-1 text-muted-foreground">
              Cached tokens come from earlier turns. Tool calls do not create them.
            </p>
          ) : null}
          <TurnMessage text={point.message} />
          <TurnToolList tools={point.turnTools} />
        </>
      ) : (
        <p className="text-muted-foreground">Hover a turn to preview it. Click a turn to read the agent message.</p>
      )}
    </div>
  );
}

function TurnMessage({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);
  const message = text.trim();
  if (!message) {
    return null;
  }

  const canCollapse = message.length > MESSAGE_COLLAPSE_LIMIT;

  return (
    <div className="mt-2 min-w-0">
      <p className={cn("min-w-0 whitespace-pre-wrap text-foreground", canCollapse && !expanded && "line-clamp-4")}>
        {message}
      </p>
      {canCollapse ? (
        <button
          type="button"
          className="mt-1 text-xs font-medium text-foreground underline-offset-2 hover:underline"
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "Collapse message" : "Expand message"}
        </button>
      ) : null}
    </div>
  );
}

function TurnToolList({ tools }: { tools: AgentTurnTool[] }) {
  if (tools.length === 0) {
    return <p className="mt-1 text-muted-foreground">This turn has no tool calls.</p>;
  }

  return (
    <ul className="mt-2 min-w-0 space-y-2 font-mono text-xs">
      {tools.map((tool, index) => (
        <TurnToolRow key={tool.id ?? `${tool.kind}-${index}`} tool={tool} />
      ))}
    </ul>
  );
}

function TurnToolRow({ tool }: { tool: AgentTurnTool }) {
  const [expanded, setExpanded] = useState(false);
  const canCollapse = tool.text.length > TOOL_TEXT_COLLAPSE_LIMIT;

  return (
    <li className="min-w-0">
      <div className="flex min-w-0 gap-2">
        <span className="shrink-0 font-semibold uppercase tracking-wide text-muted-foreground">{tool.kind}</span>
        <span
          title={tool.text}
          className={cn(
            "min-w-0 break-all",
            canCollapse && !expanded && "line-clamp-2",
            expanded && "whitespace-pre-wrap",
          )}
        >
          {tool.text}
        </span>
      </div>
      {canCollapse ? (
        <button
          type="button"
          className="mt-1 text-xs font-medium text-foreground underline-offset-2 hover:underline"
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "Collapse tool call" : "Expand tool call"}
        </button>
      ) : null}
    </li>
  );
}

function stackedBarLayout(point: AgentChartPoint, maxValue: number, plotHeight: number) {
  const inputHeight = (point.inputBar / maxValue) * plotHeight;
  const outputHeight = (point.outputBar / maxValue) * plotHeight;
  const height = Math.max(inputHeight + outputHeight, 2);
  const inputTop = PAD_TOP + plotHeight - inputHeight;
  const outputTop = inputTop - outputHeight;
  return {
    height,
    top: PAD_TOP + plotHeight - height,
    inputTop,
    outputTop,
    inputHeight,
    outputHeight,
  };
}

function turnAriaLabel(point: AgentChartPoint): string {
  return `Turn ${point.turn}: ${tokenSplitAria(point.inputBar, point.outputBar)}`;
}

function turnDetailsTitle(point: AgentChartPoint): string {
  const parts = [`Turn ${point.turn}`, ...tokenSplitLabels(point.inputBar, point.outputBar)];
  if (point.cache_read_tokens > 0) {
    parts.push(`${formatCompactTokenValue(point.cache_read_tokens)} cached`);
  }
  parts.push(toolCallLabel(point.tools));
  return parts.join(" · ");
}

function usageSummary(telemetry: AgentRunTelemetry): string {
  const parts = [
    countLabel(telemetry.num_turns, "turn", "turns"),
    countLabel(agentRunToolCallCount(telemetry), "tool call", "tool calls"),
    ...tokenSplitLabels(agentRunInputTokenCount(telemetry), agentRunOutputTokenCount(telemetry)),
  ];
  const cached = agentRunCacheReadCount(telemetry);
  if (cached > 0) {
    parts.push(`${formatCompactTokenValue(cached)} cached`);
  }
  return parts.join(" · ");
}

function tokenSplitLabels(input: number, output: number): string[] {
  const parts: string[] = [];
  if (input > 0) {
    parts.push(`${formatCompactTokenValue(input)} input`);
  }
  if (output > 0) {
    parts.push(`${formatCompactTokenValue(output)} output`);
  }
  if (parts.length === 0) {
    parts.push("0 tokens");
  }
  return parts;
}

function tokenSplitAria(input: number, output: number): string {
  const parts: string[] = [];
  if (input > 0) {
    parts.push(`${formatCompactTokenValue(input)} input tokens`);
  }
  if (output > 0) {
    parts.push(`${formatCompactTokenValue(output)} output tokens`);
  }
  if (parts.length === 0) {
    return "0 tokens";
  }
  return parts.join(", ");
}

function countLabel(count: number, singular: string, plural: string): string {
  return count === 1 ? `1 ${singular}` : `${count} ${plural}`;
}

function toolCallLabel(count: number): string {
  return countLabel(count, "tool call", "tool calls");
}
