import { LoaderCircle } from "lucide-react";
import { useMemo, useState } from "react";
import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import {
  agentRunCacheReadCount,
  agentRunInputTokenCount,
  agentRunOutputTokenCount,
  agentRunToolCallCount,
  chartPointsForTelemetry,
  type AgentChartPoint,
  type AgentRunTelemetry,
  type AgentTurnTool,
} from "@/lib/agentRunTelemetry";
import { cn } from "@/lib/utils";

type AgentRunUsageChartProps = {
  telemetry: AgentRunTelemetry;
  title?: string;
  live?: boolean;
};

const LIVE_USAGE_STATUS = "Live. The run is not finished. New turns will appear here.";
const LIVE_USAGE_DETAIL = "The run is not finished. New turns will appear here.";

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

type TokenKindFilter = "all" | "input" | "output";

export function AgentRunUsageChart({ telemetry, title, live = false }: AgentRunUsageChartProps) {
  const [tokenKind, setTokenKind] = useState<TokenKindFilter>("all");
  const [hoveredTurn, setHoveredTurn] = useState<number | null>(null);
  const [pinnedTurn, setPinnedTurn] = useState<number | null>(null);
  const points = useMemo(() => chartPointsForTelemetry(telemetry), [telemetry]);
  const activeTurn = pinnedTurn ?? hoveredTurn;
  const active = points.find((point) => point.turn === activeTurn) ?? null;

  if (telemetry.turns.length === 0) {
    return (
      <section className="min-w-0">
        {title ? <h3 className="mb-2 text-sm font-medium text-foreground">{title}</h3> : null}
        {live ? <LiveUsageStatus /> : null}
        <p className="py-2 text-sm text-muted-foreground">No usage data yet.</p>
      </section>
    );
  }

  return (
    <section className="min-w-0 overflow-hidden">
      {title ? <h3 className="mb-2 text-sm font-medium text-foreground">{title}</h3> : null}
      {live ? <LiveUsageStatus /> : null}
      <p className="mb-3 text-sm text-muted-foreground">{usageSummary(telemetry)}</p>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <TokenKindLegend value={tokenKind} onChange={setTokenKind} />
      </div>
      <div className="min-w-0" onMouseLeave={() => setHoveredTurn(null)}>
        <UsagePlot
          points={points}
          tokenKind={tokenKind}
          live={live}
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

function LiveUsageStatus() {
  return (
    <p role="status" className="mb-3 flex items-center gap-2 text-sm text-muted-foreground">
      <LoaderCircle className="size-3.5 shrink-0 animate-spin text-[color:var(--status-running-dot)]" aria-hidden />
      <span>
        <span className="font-medium text-foreground">Live.</span> {LIVE_USAGE_DETAIL}
      </span>
    </p>
  );
}

function LiveIncomingSlot({ x, plotHeight }: { x: number; plotHeight: number }) {
  const height = Math.max(plotHeight * 0.35, 16);
  return (
    <g data-testid="usage-live-slot" className="animate-pulse" pointerEvents="none">
      <rect
        x={x - 10}
        y={PAD_TOP + plotHeight - height}
        width={20}
        height={height}
        rx={3}
        className="fill-muted-foreground/20"
      />
      <text x={x} y={CHART_HEIGHT - 8} textAnchor="middle" className="fill-current text-[10px] text-muted-foreground">
        …
      </text>
    </g>
  );
}

function TokenKindLegend({ value, onChange }: { value: TokenKindFilter; onChange: (value: TokenKindFilter) => void }) {
  return (
    <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
      <TokenKindLegendButton
        label="Input"
        swatchClass={INPUT_SWATCH}
        pressed={value === "input"}
        dimmed={value === "output"}
        ariaLabel={value === "input" ? "Show input and output tokens" : "Show input tokens only"}
        onClick={() => onChange(value === "input" ? "all" : "input")}
      />
      <TokenKindLegendButton
        label="Output"
        swatchClass={OUTPUT_SWATCH}
        pressed={value === "output"}
        dimmed={value === "input"}
        ariaLabel={value === "output" ? "Show input and output tokens" : "Show output tokens only"}
        onClick={() => onChange(value === "output" ? "all" : "output")}
      />
    </div>
  );
}

function TokenKindLegendButton({
  label,
  swatchClass,
  pressed,
  dimmed,
  ariaLabel,
  onClick,
}: {
  label: string;
  swatchClass: string;
  pressed: boolean;
  dimmed: boolean;
  ariaLabel: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      aria-label={ariaLabel}
      className={cn(
        "inline-flex items-center gap-1.5 rounded px-0.5 font-medium hover:text-foreground",
        pressed ? "text-foreground" : "text-muted-foreground",
        dimmed && "opacity-40",
      )}
      onClick={onClick}
    >
      <span className={cn("size-2.5 rounded-sm", swatchClass)} />
      {label}
    </button>
  );
}

function UsagePlot({
  points,
  tokenKind,
  live,
  hoveredTurn,
  pinnedTurn,
  onHoverTurn,
  onPinTurn,
}: {
  points: AgentChartPoint[];
  tokenKind: TokenKindFilter;
  live: boolean;
  hoveredTurn: number | null;
  pinnedTurn: number | null;
  onHoverTurn: (turn: number | null) => void;
  onPinTurn: (turn: number | null) => void;
}) {
  const plotHeight = CHART_HEIGHT - PAD_TOP - PAD_BOTTOM;
  const slotCount = points.length + (live ? 1 : 0);
  const chartWidth = Math.max(640, PAD_LEFT + PAD_RIGHT + slotCount * BAR_SLOT);
  const plotWidth = chartWidth - PAD_LEFT - PAD_RIGHT;
  const maxValue = Math.max(1, ...points.map((point) => visibleBarTotal(point, tokenKind)));
  const step = slotCount > 1 ? plotWidth / (slotCount - 1) : 0;
  const columnX = (index: number) => PAD_LEFT + (slotCount === 1 ? plotWidth / 2 : index * step);

  return (
    <div className="overflow-x-auto">
      <svg
        role="img"
        aria-label={chartAriaLabel(tokenKind, live)}
        viewBox={`0 0 ${chartWidth} ${CHART_HEIGHT}`}
        className="h-52 w-full min-w-[40rem]"
      >
        <text x={4} y={PAD_TOP + 10} className="fill-current text-[10px] text-muted-foreground" pointerEvents="none">
          {formatCompactTokenValue(maxValue)}
        </text>
        {points.map((point, index) => {
          const x = columnX(index);
          const stack = stackedBarLayout(point, maxValue, plotHeight, tokenKind);
          const hovered = hoveredTurn === point.turn;
          const pinned = pinnedTurn === point.turn;
          return (
            <g key={point.turn} pointerEvents="none">
              {stack.inputHeight > 0 ? (
                <rect
                  data-testid="usage-bar-input"
                  x={x - 10}
                  y={stack.inputTop}
                  width={20}
                  height={Math.max(stack.inputHeight, 1)}
                  className={INPUT_BAR_FILL}
                />
              ) : null}
              {stack.outputHeight > 0 ? (
                <rect
                  data-testid="usage-bar-output"
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
              aria-label={turnAriaLabel(point, tokenKind)}
              className="cursor-pointer fill-transparent"
              pointerEvents="all"
              onMouseEnter={() => onHoverTurn(point.turn)}
              onClick={() => onPinTurn(pinned ? null : point.turn)}
            />
          );
        })}
        {live ? <LiveIncomingSlot x={columnX(points.length)} plotHeight={plotHeight} /> : null}
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

function stackedBarLayout(point: AgentChartPoint, maxValue: number, plotHeight: number, tokenKind: TokenKindFilter) {
  const inputBar = tokenKind === "output" ? 0 : point.inputBar;
  const outputBar = tokenKind === "input" ? 0 : point.outputBar;
  const inputHeight = (inputBar / maxValue) * plotHeight;
  const outputHeight = (outputBar / maxValue) * plotHeight;
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

function visibleBarTotal(point: AgentChartPoint, tokenKind: TokenKindFilter): number {
  if (tokenKind === "input") {
    return point.inputBar;
  }
  if (tokenKind === "output") {
    return point.outputBar;
  }
  return point.inputBar + point.outputBar;
}

function chartAriaLabel(tokenKind: TokenKindFilter, live: boolean): string {
  let base = "Input and output tokens by turn";
  if (tokenKind === "input") {
    base = "Input tokens by turn";
  } else if (tokenKind === "output") {
    base = "Output tokens by turn";
  }
  if (!live) {
    return base;
  }
  return `${base}. ${LIVE_USAGE_STATUS}`;
}

function turnAriaLabel(point: AgentChartPoint, tokenKind: TokenKindFilter): string {
  const input = tokenKind === "output" ? 0 : point.inputBar;
  const output = tokenKind === "input" ? 0 : point.outputBar;
  return `Turn ${point.turn}: ${tokenSplitAria(input, output)}`;
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
