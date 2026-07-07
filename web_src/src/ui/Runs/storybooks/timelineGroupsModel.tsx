import JsonView from "@uiw/react-json-view";
import { ChevronDown, ChevronRight, GitBranch, Pencil, Terminal, Webhook } from "lucide-react";
import { useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";
import { RunNodeDetailDetailsView } from "../RunNodeDetailDetailsView";
import {
  CardMarker,
  EventRail,
  EventStatusPill,
  InputChainModal,
  PayloadEventCard,
  type InputChainStep,
} from "../RunStepTimeline";
import { DetailBox, ErrorDetailBox, HeaderIconButton } from "../RunStepAccordion";
import { RunStepConfigFields } from "../RunStepConfigView";

/** Real node fed into the Runtime Config card so its read-only form has schema/values. */
export interface RuntimeConfigNode {
  component?: string;
  name?: string;
  configuration?: Record<string, unknown>;
  iconSrc?: string;
  iconSlug?: string;
}

/**
 * Wireframe-only model for the flat "timeline events" design (never merged to
 * production). The feed is a single list of events on one rail, each rendered
 * either as a Card (rich, GitHub-comment style) or a Line (compact, GitHub-commit
 * style, sometimes expandable to reveal a comment/detail). No lifecycle grouping.
 */

export interface TimelineActor {
  /** GitHub username; used both as the display name and to fetch the avatar. */
  name: string;
  initials?: string;
}

export interface NetworkExchange {
  method: "GET" | "POST" | "PATCH" | "DELETE";
  url: string;
  status: number;
  statusText?: string;
  duration?: string;
  requestHeaders?: Record<string, string>;
  requestBody?: unknown;
  responseHeaders?: Record<string, string>;
  responseBody?: unknown;
}

export interface TimelineLine {
  id: string;
  /** Tailwind bg-* class for the rail marker dot. */
  dotClassName: string;
  label: ReactNode;
  timestamp?: string;
  actor?: TimelineActor;
  /** When present, the line renders Network-tab style and expands to a Headers/Payload/Response detail. */
  request?: NetworkExchange;
  /** When present (and no `request`), the line is expandable to reveal this comment/reason box. */
  detail?: ReactNode;
}

export type TimelineCardEvent =
  | {
      kind: "payload";
      kicker?: string;
      status: { dotClassName: string; label: string };
      sourceName: string;
      sourceTrailing?: ReactNode;
      meta?: string | null;
      payload: unknown;
      nodeName: string;
      nodeIcon: ReactNode;
    }
  | {
      kind: "summary";
      status: { badgeColor: string; label: string };
      relativeTime?: string;
      details: Record<string, unknown>;
    }
  | {
      kind: "logs";
      status: { dotClassName: string; label: string };
      sourceName: string;
      meta?: string | null;
      lines: string[];
    }
  | {
      kind: "error";
      message?: string;
      reason?: string;
      metadata?: Record<string, unknown>;
    };

export type TimelineEvent =
  | { type: "card"; id: string; card: TimelineCardEvent }
  | { type: "line"; id: string; line: TimelineLine };

/** Mocked upstream steps shown by the input-chain modal in the wireframe. Most recent first. */
const MOCK_INPUT_CHAIN: InputChainStep[] = [
  {
    nodeId: "build-and-test",
    name: "build-and-test",
    icon: <Terminal className="h-3.5 w-3.5" />,
    payload: { status: "passed", tests_passed: 42, duration_seconds: 52 },
  },
  {
    nodeId: "checkout-code",
    name: "checkout-code",
    icon: <GitBranch className="h-3.5 w-3.5" />,
    payload: { ref: "main", sha: "9f3c1a2", trigger: "push" },
  },
  {
    nodeId: "on-push",
    name: "on-push",
    icon: <Webhook className="h-3.5 w-3.5" />,
    payload: { repository: "acme/store", pusher: "ci-bot", branch: "main" },
  },
];

/** The "+X more" chip on an Input card; opens the input-chain modal (wireframe, mocked steps). */
export function InputChainMoreChip({ count, steps = MOCK_INPUT_CHAIN }: { count: number; steps?: InputChainStep[] }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        title="Open input chain"
        onClick={(event) => {
          event.stopPropagation();
          setOpen(true);
        }}
        className="flex shrink-0 items-center rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 transition-colors hover:bg-slate-200 hover:text-slate-700"
      >
        +{count} more
      </button>
      <InputChainModal open={open} onOpenChange={setOpen} steps={steps} />
    </>
  );
}

/** Rail marker for a small line event: a colored dot centered in the rail column. */
function DotMarker({ className }: { className: string }) {
  return (
    <span className="flex h-6 w-6 shrink-0 items-center justify-center">
      <span className={cn("h-2.5 w-2.5 rounded-full", className)} />
    </span>
  );
}

/** A terminal-styled logs card (static output for the wireframe). */
function LogsCard({ card }: { card: Extract<TimelineCardEvent, { kind: "logs" }> }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="overflow-hidden rounded border border-slate-200 bg-white">
      <div className="flex items-center gap-1.5 border-b border-slate-200 bg-slate-50 px-3 py-1.5">
        <EventStatusPill dotClassName={card.status.dotClassName} label={card.status.label} />
        <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-400">Logs</span>
        <span className="min-w-0 truncate text-[12px] font-medium text-slate-600">{card.sourceName}</span>
        <div className="ml-auto flex shrink-0 items-center gap-0.5">
          {card.meta ? <span className="pr-1 text-[11px] tabular-nums text-slate-600">{card.meta}</span> : null}
          <HeaderIconButton
            label={open ? "Collapse" : "Expand"}
            icon={open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            active={open}
            onClick={() => setOpen((value) => !value)}
          />
        </div>
      </div>
      {open ? (
        <pre className="max-h-64 overflow-auto bg-slate-900 px-3 py-2.5 font-mono text-[11px] leading-relaxed text-slate-100">
          {card.lines.join("\n")}
        </pre>
      ) : null}
    </div>
  );
}

/** Payload card that starts collapsed (header only); the parent-controlled open state defers expansion to a click. */
function CollapsedPayloadCard({ card }: { card: Extract<TimelineCardEvent, { kind: "payload" }> }) {
  const [open, setOpen] = useState(false);
  return (
    <PayloadEventCard
      kicker={card.kicker}
      status={card.status}
      sourceName={card.sourceName}
      sourceTrailing={card.sourceTrailing}
      meta={card.meta}
      open={open}
      onToggleOpen={() => setOpen((value) => !value)}
      payload={card.payload}
      modalNodeName={card.nodeName}
      modalNodeIcon={card.nodeIcon}
    />
  );
}

/** Small segmented control on the Runtime Config card: read-only form vs raw JSON. */
function ConfigViewToggle({ view, onChange }: { view: "form" | "json"; onChange: (view: "form" | "json") => void }) {
  const options: { id: "form" | "json"; label: string }[] = [
    { id: "form", label: "Form" },
    { id: "json", label: "JSON" },
  ];
  return (
    <div className="flex shrink-0 items-center gap-0.5 rounded-md border border-slate-200 bg-white p-0.5">
      {options.map((option) => {
        const active = view === option.id;
        return (
          <button
            key={option.id}
            type="button"
            onClick={() => onChange(option.id)}
            aria-pressed={active}
            className={cn(
              "rounded px-1.5 py-0.5 text-[10px] font-medium transition-colors",
              active ? "bg-slate-100 text-slate-800" : "text-slate-500 hover:text-slate-700",
            )}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}

/**
 * The Runtime Config card: renders a step's configuration as a read-only form (default)
 * or raw JSON, with a per-step toggle and an Edit button (wireframe placeholder for now,
 * intended to later enter edit mode with this node selected). Collapsed by default like
 * the other cards. Falls back to JSON-only when no node schema is available.
 */
function RuntimeConfigCard({
  card,
  configNode,
}: {
  card: Extract<TimelineCardEvent, { kind: "payload" }>;
  configNode?: RuntimeConfigNode;
}) {
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<"form" | "json">("form");
  const canShowForm = !!configNode;
  const effectiveView = canShowForm ? view : "json";
  return (
    <div className="overflow-hidden rounded border border-slate-200 bg-white">
      <div className="flex items-center gap-1.5 border-b border-slate-200 bg-slate-50 px-3 py-1.5">
        <EventStatusPill dotClassName={card.status.dotClassName} label={card.status.label} />
        <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-400">
          Runtime Config
        </span>
        <span className="min-w-0 truncate text-[12px] font-medium text-slate-600">{card.sourceName}</span>
        <div className="ml-auto flex shrink-0 items-center gap-1.5">
          {card.meta ? <span className="pr-1 text-[11px] tabular-nums text-slate-600">{card.meta}</span> : null}
          {open && canShowForm ? <ConfigViewToggle view={view} onChange={setView} /> : null}
          {open ? <HeaderIconButton label="Edit configuration" icon={<Pencil className="h-3.5 w-3.5" />} /> : null}
          <HeaderIconButton
            label={open ? "Collapse" : "Expand"}
            icon={open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            active={open}
            onClick={() => setOpen((value) => !value)}
          />
        </div>
      </div>
      {open ? (
        <div className="px-3 py-2.5">
          {effectiveView === "form" ? (
            <RunStepConfigFields component={configNode?.component} configuration={configNode?.configuration} />
          ) : (
            <JsonView
              value={(card.payload ?? {}) as object}
              collapsed={2}
              displayDataTypes={false}
              style={{ fontSize: 12 }}
            />
          )}
        </div>
      ) : null}
    </div>
  );
}

function CardEventView({ card, configNode }: { card: TimelineCardEvent; configNode?: RuntimeConfigNode }) {
  if (card.kind === "payload") {
    if (card.kicker === "Runtime Config") {
      return <RuntimeConfigCard card={card} configNode={configNode} />;
    }
    return <CollapsedPayloadCard card={card} />;
  }
  if (card.kind === "summary") {
    return (
      <DetailBox title="Summary">
        <RunNodeDetailDetailsView details={card.details} statusBadge={card.status} relativeTime={card.relativeTime} />
      </DetailBox>
    );
  }
  if (card.kind === "error") {
    return <ErrorDetailBox message={card.message} reason={card.reason} metadata={card.metadata} />;
  }
  return <LogsCard card={card} />;
}

/** Monospace HTTP method badge (GET/POST/...). */
function MethodChip({ method }: { method: NetworkExchange["method"] }) {
  return (
    <span className="shrink-0 rounded border border-slate-200 bg-white px-1 py-px font-mono text-[10px] font-semibold tracking-wide text-slate-600">
      {method}
    </span>
  );
}

/** Colored HTTP status badge (2xx emerald, otherwise red). */
function StatusChip({ status, statusText }: { status: number; statusText?: string }) {
  const ok = status >= 200 && status < 300;
  return (
    <span
      className={cn(
        "shrink-0 rounded px-1.5 py-px font-mono text-[10px] font-semibold tabular-nums ring-1",
        ok ? "bg-emerald-50 text-emerald-700 ring-emerald-200" : "bg-red-50 text-red-700 ring-red-200",
      )}
    >
      {status}
      {statusText ? ` ${statusText}` : ""}
    </span>
  );
}

function HeaderList({ headers }: { headers?: Record<string, string> }) {
  const entries = headers ? Object.entries(headers) : [];
  if (entries.length === 0) return <span className="text-[11px] italic text-slate-400">None</span>;
  return (
    <div className="flex flex-col gap-0.5">
      {entries.map(([key, value]) => (
        <div key={key} className="flex gap-2 font-mono text-[11px] leading-snug">
          <span className="shrink-0 text-slate-500">{key}:</span>
          <span className="min-w-0 break-all text-slate-700">{value}</span>
        </div>
      ))}
    </div>
  );
}

function DetailSection({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-400">{label}</span>
      {children}
    </div>
  );
}

/** Chrome-DevTools-style detail for a network exchange: Headers / Payload / Response tabs. */
function NetworkExchangeDetail({ exchange }: { exchange: NetworkExchange }) {
  const [tab, setTab] = useState<"headers" | "payload" | "response">("headers");
  const tabs: { id: typeof tab; label: string }[] = [
    { id: "headers", label: "Headers" },
    { id: "payload", label: "Payload" },
    { id: "response", label: "Response" },
  ];
  return (
    <div className="mt-1 overflow-hidden rounded border border-slate-200 bg-white">
      <div className="flex items-center gap-0.5 border-b border-slate-200 bg-slate-50 px-1.5 py-1">
        {tabs.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => setTab(item.id)}
            className={cn(
              "rounded px-2 py-0.5 text-[11px] font-medium transition-colors",
              tab === item.id ? "bg-white text-slate-800 ring-1 ring-slate-200" : "text-slate-500 hover:text-slate-700",
            )}
          >
            {item.label}
          </button>
        ))}
      </div>
      <div className="flex flex-col gap-3 px-3 py-2.5">
        {tab === "headers" ? (
          <>
            <DetailSection label="General">
              <div className="flex flex-col gap-0.5 font-mono text-[11px] leading-snug">
                <div className="flex gap-2">
                  <span className="shrink-0 text-slate-500">Request URL:</span>
                  <span className="min-w-0 break-all text-slate-700">{exchange.url}</span>
                </div>
                <div className="flex gap-2">
                  <span className="shrink-0 text-slate-500">Request Method:</span>
                  <span className="text-slate-700">{exchange.method}</span>
                </div>
                <div className="flex gap-2">
                  <span className="shrink-0 text-slate-500">Status Code:</span>
                  <span className="text-slate-700">
                    {exchange.status}
                    {exchange.statusText ? ` ${exchange.statusText}` : ""}
                  </span>
                </div>
              </div>
            </DetailSection>
            <DetailSection label="Request Headers">
              <HeaderList headers={exchange.requestHeaders} />
            </DetailSection>
            <DetailSection label="Response Headers">
              <HeaderList headers={exchange.responseHeaders} />
            </DetailSection>
          </>
        ) : null}
        {tab === "payload" ? (
          exchange.requestBody !== undefined ? (
            <JsonView
              value={exchange.requestBody as object}
              collapsed={2}
              displayDataTypes={false}
              style={{ fontSize: 12 }}
            />
          ) : (
            <span className="text-[11px] italic text-slate-400">No request payload</span>
          )
        ) : null}
        {tab === "response" ? (
          exchange.responseBody !== undefined ? (
            <JsonView
              value={exchange.responseBody as object}
              collapsed={2}
              displayDataTypes={false}
              style={{ fontSize: 12 }}
            />
          ) : (
            <span className="text-[11px] italic text-slate-400">No response body</span>
          )
        ) : null}
      </div>
    </div>
  );
}

/** A compact, GitHub-commit-style line event; expandable to a comment or a network detail. */
function LineEventRow({ line }: { line: TimelineLine }) {
  const [open, setOpen] = useState(false);
  const network = line.request;
  const expandable = !!network || !!line.detail;
  return (
    <div>
      <div className="flex min-h-6 items-center gap-2">
        {network ? <MethodChip method={network.method} /> : null}
        {line.actor ? (
          <img
            src={`https://github.com/${line.actor.name}.png?size=40`}
            alt={line.actor.name}
            title={line.actor.name}
            className="h-5 w-5 shrink-0 rounded-full bg-slate-200 object-cover ring-1 ring-slate-200"
          />
        ) : null}
        <span className="min-w-0 text-[12px] text-slate-700">{line.label}</span>
        <div className="ml-auto flex shrink-0 items-center gap-1.5">
          {network ? <StatusChip status={network.status} statusText={network.statusText} /> : null}
          {network?.duration ? (
            <span className="text-[11px] tabular-nums text-slate-500">{network.duration}</span>
          ) : null}
          {line.timestamp ? <span className="text-[11px] tabular-nums text-slate-500">{line.timestamp}</span> : null}
          {expandable ? (
            <HeaderIconButton
              label={open ? "Collapse" : "Expand"}
              icon={open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
              active={open}
              onClick={() => setOpen((value) => !value)}
            />
          ) : null}
        </div>
      </div>
      {open && network ? <NetworkExchangeDetail exchange={network} /> : null}
      {open && !network && line.detail ? (
        <div className="mt-1 rounded border border-slate-200 bg-white px-3 py-2 text-[12px] leading-snug text-slate-600">
          {line.detail}
        </div>
      ) : null}
    </div>
  );
}

/**
 * Lucide `square-arrow-right-enter` / `square-arrow-right-exit` (added after our installed
 * lucide-react version), inlined here for the wireframe to mark Input (triggered) and
 * Output (finished) events without upgrading the shared dependency.
 */
function SquareArrowRightEnterIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      <path d="m10 16 4-4-4-4" />
      <path d="M3 12h11" />
      <path d="M3 8V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3" />
    </svg>
  );
}

function SquareArrowRightExitIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      <path d="M10 12h11" />
      <path d="m17 16 4-4-4-4" />
      <path d="M21 6.344V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-1.344" />
    </svg>
  );
}

/** Rail marker for a card event: an icon (payload) or a status dot (summary/logs) in a ring. */
function cardMarker(card: TimelineCardEvent): ReactNode {
  if (card.kind === "payload") {
    // Input (triggered) and Output (finished) events read as enter/exit arrows; other
    // payload cards (e.g. Runtime Config) keep the component icon.
    const icon =
      card.kicker === "Input" ? (
        <SquareArrowRightEnterIcon className="h-3.5 w-3.5" />
      ) : card.kicker === "Output" ? (
        <SquareArrowRightExitIcon className="h-3.5 w-3.5" />
      ) : (
        card.nodeIcon
      );
    return <CardMarker>{icon}</CardMarker>;
  }
  if (card.kind === "error") {
    return (
      <CardMarker>
        <span className="h-2.5 w-2.5 rounded-full bg-red-500" />
      </CardMarker>
    );
  }
  const dotClassName = card.kind === "summary" ? card.status.badgeColor : card.status.dotClassName;
  return (
    <CardMarker>
      <span className={cn("h-2.5 w-2.5 rounded-full", dotClassName)} />
    </CardMarker>
  );
}

/** A single flat feed row: marker column + card/line content. */
function EventRow({
  event,
  isLast,
  configNode,
}: {
  event: TimelineEvent;
  isLast?: boolean;
  configNode?: RuntimeConfigNode;
}) {
  if (event.type === "line") {
    return (
      <EventRail marker={<DotMarker className={event.line.dotClassName} />} isLast={isLast}>
        <LineEventRow line={event.line} />
      </EventRail>
    );
  }
  return (
    <EventRail marker={cardMarker(event.card)} isLast={isLast}>
      <CardEventView card={event.card} configNode={configNode} />
    </EventRail>
  );
}

/** Line-event ids for queue lifecycle transitions, which we don't surface in the timeline. */
const QUEUE_LINE_IDS = new Set(["q-enter", "q-exit"]);

export function EventTimeline({ events, configNode }: { events: TimelineEvent[]; configNode?: RuntimeConfigNode }) {
  // Queueing (enter/exit queue) is noise for this view, so it's dropped from the feed.
  const visible = events.filter((event) => !(event.type === "line" && QUEUE_LINE_IDS.has(event.id)));
  // The run summary is pinned to the top (before the timeline), not left as a terminal
  // feed item; everything else renders as the flat event feed below it.
  const summary = visible.find(
    (event): event is Extract<TimelineEvent, { type: "card" }> =>
      event.type === "card" && event.card.kind === "summary",
  );
  const feed = summary ? visible.filter((event) => event !== summary) : visible;

  return (
    <div className="bg-slate-50 px-3 py-3">
      {summary ? (
        <div className="mb-3">
          <CardEventView card={summary.card} />
        </div>
      ) : null}
      {feed.map((event, index) => (
        <EventRow key={event.id} event={event} isLast={index === feed.length - 1} configNode={configNode} />
      ))}
    </div>
  );
}
