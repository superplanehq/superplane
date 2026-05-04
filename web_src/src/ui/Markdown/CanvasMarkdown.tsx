import React, { Children, isValidElement } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import {
  Info,
  AlertTriangle,
  OctagonAlert,
  Lightbulb,
  MessageCircleWarning,
  ExternalLink,
  CircleHelp,
  Play,
} from "lucide-react";

import { resolveIcon } from "@/lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";

import { MermaidDiagram } from "./MermaidDiagram";
import { WidgetBlock } from "./WidgetBlock";

import "highlight.js/styles/github.css";

//
// Node-reference chips: `@slug` and `[[node:slug]]` in markdown render as an
// inline chip. When the slug matches a known node, the chip navigates to the
// canvas with that node selected (?node=<slug>).
//
// Detection happens on the mdast before rehype, so matches inside inline code
// and code blocks are left alone — those nodes never have their children
// visited.
//

export const NODE_REF_CLASS = "sp-node-ref";
//
// Trigger-run chip: a sibling token to the node-ref chip that targets a
// specific template on a Manual Run trigger. Authored as either
// `@trigger:run/template-name` or `[[run:trigger:template-name]]`. When the
// surrounding NodeChipContext provides `onTriggerTemplateRun`, the chip
// becomes a button that opens the existing EmitEventModal pre-filled with the
// template payload; otherwise it renders disabled.
//
export const TRIGGER_RUN_CLASS = "sp-trigger-run";
//
// Node-status chip: a sibling token to the node-ref chip that renders the
// latest execution status badge for a (component / blueprint / widget) node.
// Authored as either `@node:status` or `[[status:node]]`. Always rendered as
// a non-interactive `<span>`; navigation is handled by the regular
// `@node-name` chip.
//
export const NODE_STATUS_CLASS = "sp-node-status";

// Order matters: longer / more-specific patterns first so the regex prefers
// the run / status variants over the bare node reference when both could
// match.
const NODE_TOKEN_RE =
  /(\[\[run:([a-zA-Z0-9][a-zA-Z0-9_-]*):([a-zA-Z0-9][a-zA-Z0-9_-]*)\]\]|\[\[node:([a-zA-Z0-9][a-zA-Z0-9_-]*)\]\]|\[\[status:([a-zA-Z0-9][a-zA-Z0-9_-]*)\]\]|(?<![A-Za-z0-9_])@([a-zA-Z][a-zA-Z0-9_-]*):run\/([a-zA-Z0-9][a-zA-Z0-9_-]*)|(?<![A-Za-z0-9_])@([a-zA-Z][a-zA-Z0-9_-]*):status\b|(?<![A-Za-z0-9_])@([a-zA-Z][a-zA-Z0-9_-]*))/g;

type MdastNode = {
  type: string;
  value?: string;
  children?: MdastNode[];
};

function escapeHtml(input: string): string {
  return input
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

export function remarkNodeRefs() {
  return (tree: MdastNode) => {
    const visit = (nodes: MdastNode[]) => {
      for (let i = 0; i < nodes.length; i++) {
        const node = nodes[i];

        // Don't descend into code / inlineCode — @slug inside backticks is
        // literal.
        if (node.type === "code" || node.type === "inlineCode") continue;

        if (node.type === "text" && typeof node.value === "string") {
          const value = node.value;
          NODE_TOKEN_RE.lastIndex = 0;
          if (!NODE_TOKEN_RE.test(value)) continue;
          NODE_TOKEN_RE.lastIndex = 0;

          const replacements: MdastNode[] = [];
          let lastIndex = 0;
          let match: RegExpExecArray | null;
          while ((match = NODE_TOKEN_RE.exec(value)) !== null) {
            const full = match[0];
            // Capture groups in the regex (1-indexed):
            //  2,3  -> [[run:trigger:template]]
            //  4    -> [[node:slug]]
            //  5    -> [[status:slug]]
            //  6,7  -> @trigger:run/template
            //  8    -> @slug:status
            //  9    -> @slug
            const runTrigger = match[2] || match[6];
            const runTemplate = match[3] || match[7];
            const statusSlug = match[5] || match[8];
            const nodeSlug = match[4] || match[9];

            if (match.index > lastIndex) {
              replacements.push({ type: "text", value: value.slice(lastIndex, match.index) });
            }
            if (runTrigger && runTemplate) {
              replacements.push({
                type: "html",
                value: `<span class="${TRIGGER_RUN_CLASS}" data-trigger="${escapeHtml(runTrigger)}" data-template="${escapeHtml(runTemplate)}">${escapeHtml(`${runTrigger}:run/${runTemplate}`)}</span>`,
              });
            } else if (statusSlug) {
              replacements.push({
                type: "html",
                value: `<span class="${NODE_STATUS_CLASS}" data-node="${escapeHtml(statusSlug)}">${escapeHtml(`${statusSlug}:status`)}</span>`,
              });
            } else if (nodeSlug) {
              replacements.push({
                type: "html",
                value: `<span class="${NODE_REF_CLASS}">${escapeHtml(nodeSlug)}</span>`,
              });
            }
            lastIndex = match.index + full.length;
          }
          if (lastIndex < value.length) {
            replacements.push({ type: "text", value: value.slice(lastIndex) });
          }

          nodes.splice(i, 1, ...replacements);
          i += replacements.length - 1;
          continue;
        }

        if (Array.isArray(node.children)) {
          visit(node.children);
        }
      }
    };

    if (Array.isArray(tree.children)) {
      visit(tree.children);
    }
  };
}

//
// Allow <details>/<summary> (collapsible sections) to survive sanitization,
// on top of the default-safe rehype-sanitize schema. Also allow className on
// span so the node-ref chip marker survives.
//
const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary"],
  attributes: {
    ...(defaultSchema.attributes ?? {}),
    span: [
      ...((defaultSchema.attributes ?? {}).span ?? []),
      "className",
      "class",
      "data-trigger",
      "data-template",
      "data-node",
    ],
  },
};

//
// Inline badge convention: `type:label` renders as a colored pill. Anything
// else renders as regular inline code. Authors use this to show statuses
// and inline labels without inventing custom markdown syntax.
//

const BADGE_COLORS: Record<string, string> = {
  status: "bg-emerald-100 text-emerald-700 border-emerald-200",
  success: "bg-emerald-100 text-emerald-700 border-emerald-200",
  warning: "bg-amber-100 text-amber-700 border-amber-200",
  error: "bg-red-100 text-red-700 border-red-200",
  info: "bg-blue-100 text-blue-700 border-blue-200",
  duration: "bg-slate-100 text-slate-600 border-slate-200",
};

const BADGE_RE = /^(status|success|warning|error|info|duration):(.+)$/;

function extractCodeString(children: React.ReactNode): string {
  let text = "";
  Children.forEach(children, (child) => {
    if (typeof child === "string") {
      text += child;
    } else if (typeof child === "number") {
      text += String(child);
    } else if (isValidElement<{ children?: React.ReactNode }>(child) && child.props.children != null) {
      text += extractCodeString(child.props.children);
    }
  });
  return text;
}

interface InlineCodeContext {
  canvasId?: string;
  nodeRefs?: NodeChipContext;
}

function buildInlineCodeComponent({ canvasId, nodeRefs }: InlineCodeContext) {
  return function InlineCode({ children, className }: { children?: React.ReactNode; className?: string }) {
    const classes = typeof className === "string" ? className : "";

    // Fenced ```mermaid block → render as a live diagram instead of highlighted code.
    if (/(^|\s)language-mermaid(\s|$)/.test(classes)) {
      const source = extractCodeString(children).replace(/\n$/, "");
      return <MermaidDiagram code={source} />;
    }

    // Fenced ```widget block (or the deprecated ```query alias) → render as a
    // table / chart / number widget when canvasId is available. In surfaces
    // that don't supply canvasId (e.g. Readme modal) the block falls through
    // to the regular highlighted code rendering.
    if ((/(^|\s)language-widget(\s|$)/.test(classes) || /(^|\s)language-query(\s|$)/.test(classes)) && canvasId) {
      const source = extractCodeString(children).replace(/\n$/, "");
      return <WidgetBlock body={source} canvasId={canvasId} nodeRefs={nodeRefs} />;
    }

    const text = String(children ?? "");
    const match = text.match(BADGE_RE);

    if (match) {
      const [, type, label] = match;
      const color = BADGE_COLORS[type] ?? BADGE_COLORS.info;
      return (
        <span
          className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium leading-none ${color}`}
        >
          {label}
        </span>
      );
    }

    //
    // Fenced code blocks arrive with `className="language-xxx"` and are already
    // wrapped in a <pre> by react-markdown — we pass those through untouched so
    // rehype-highlight keeps control of their styling. Inline snippets (no
    // language-* class) get a pill treatment so short commands / paths /
    // identifiers stand out from the surrounding prose.
    //
    const isFencedBlock = /(^|\s)language-/.test(classes);
    if (isFencedBlock) {
      return <code className={className}>{children}</code>;
    }

    return (
      <code className="rounded border border-slate-300 bg-slate-100 px-1.5 py-0.5 font-mono text-[0.85em] text-slate-800">
        {children}
      </code>
    );
  };
}

//
// GitHub-style admonitions written as `> [!NOTE]` blockquote prefixes.
//

type AdmonitionType = "NOTE" | "TIP" | "IMPORTANT" | "WARNING" | "CAUTION";

const ADMONITION_CONFIG: Record<
  AdmonitionType,
  { icon: React.FC<{ className?: string }>; border: string; bg: string; title: string; titleColor: string }
> = {
  NOTE: { icon: Info, border: "border-blue-300", bg: "bg-blue-50", title: "Note", titleColor: "text-blue-700" },
  TIP: {
    icon: Lightbulb,
    border: "border-emerald-300",
    bg: "bg-emerald-50",
    title: "Tip",
    titleColor: "text-emerald-700",
  },
  IMPORTANT: {
    icon: MessageCircleWarning,
    border: "border-purple-300",
    bg: "bg-purple-50",
    title: "Important",
    titleColor: "text-purple-700",
  },
  WARNING: {
    icon: AlertTriangle,
    border: "border-amber-300",
    bg: "bg-amber-50",
    title: "Warning",
    titleColor: "text-amber-700",
  },
  CAUTION: {
    icon: OctagonAlert,
    border: "border-red-300",
    bg: "bg-red-50",
    title: "Caution",
    titleColor: "text-red-700",
  },
};

const ADMONITION_DETECT_RE = /\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]/;
const ADMONITION_STRIP_RE = /\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/;

function extractTextFromChildren(children: React.ReactNode): string {
  let text = "";
  Children.forEach(children, (child) => {
    if (typeof child === "string") {
      text += child;
    } else if (isValidElement<{ children?: React.ReactNode }>(child) && child.props.children) {
      text += extractTextFromChildren(child.props.children);
    }
  });
  return text;
}

function stripAdmonitionTag(children: React.ReactNode): React.ReactNode {
  let stripped = false;

  function processNode(node: React.ReactNode): React.ReactNode {
    if (stripped) return node;

    if (typeof node === "string") {
      const replaced = node.replace(ADMONITION_STRIP_RE, "");
      if (replaced !== node) {
        stripped = true;
        const trimmed = replaced.replace(/^\n+/, "");
        return trimmed || null;
      }
      return node;
    }

    if (isValidElement<{ children?: React.ReactNode }>(node) && node.props.children != null) {
      const newChildren = Children.map(node.props.children, (c) => processNode(c));
      if (stripped) {
        return React.cloneElement(node as React.ReactElement<{ children?: React.ReactNode }>, {}, newChildren);
      }
    }

    return node;
  }

  return Children.map(children, (child) => processNode(child));
}

function Blockquote({ children }: { children?: React.ReactNode }) {
  const rawText = extractTextFromChildren(children);
  const match = rawText.match(ADMONITION_DETECT_RE);

  if (match) {
    const type = match[1] as AdmonitionType;
    const config = ADMONITION_CONFIG[type];
    const Icon = config.icon;
    const strippedChildren = stripAdmonitionTag(children);

    return (
      <div className={`my-2 rounded-md border-l-4 ${config.border} ${config.bg} p-3`}>
        <div className={`mb-1 flex items-center gap-1.5 text-xs font-semibold ${config.titleColor}`}>
          <Icon className="h-3.5 w-3.5" />
          {config.title}
        </div>
        <div className="text-sm text-gray-700 [&_p]:my-0.5">{strippedChildren}</div>
      </div>
    );
  }

  return <blockquote className="my-2 border-l-2 border-slate-300 pl-3 text-gray-600">{children}</blockquote>;
}

function Anchor({ href, children }: { href?: string; children?: React.ReactNode }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-0.5 text-blue-600 underline"
    >
      {children}
      <ExternalLink className="inline h-3 w-3 shrink-0" />
    </a>
  );
}

function Table({ children }: { children?: React.ReactNode }) {
  return (
    <div className="my-2 overflow-x-auto rounded border border-slate-200">
      <table className="min-w-full border-collapse text-left text-xs">{children}</table>
    </div>
  );
}

function Th({ children }: { children?: React.ReactNode }) {
  return (
    <th className="border-b border-slate-200 bg-slate-50 px-3 py-1.5 text-xs font-semibold text-gray-600">
      {children}
    </th>
  );
}

function Td({ children }: { children?: React.ReactNode }) {
  return <td className="border-b border-slate-100 px-3 py-1.5">{children}</td>;
}

function Img({ src, alt }: { src?: string; alt?: string }) {
  return <img src={src} alt={alt ?? ""} className="my-2 max-h-64 rounded-lg border border-slate-200" />;
}

function Hr() {
  return <hr className="my-3 border-slate-200" />;
}

function Details({ children }: { children?: React.ReactNode }) {
  const childArray = Children.toArray(children);
  const summary = childArray.find((c) => isValidElement(c) && (c.type === Summary || c.type === "summary"));
  const body = childArray.filter((c) => c !== summary);

  return (
    <details className="my-2 rounded-md border border-slate-200 bg-white text-sm [&[open]>summary]:border-b [&[open]>summary]:border-slate-200">
      {summary}
      {body.length > 0 && <DetailsContent>{body}</DetailsContent>}
    </details>
  );
}

function Summary({ children }: { children?: React.ReactNode }) {
  // The native disclosure marker renders right at the start of the summary's
  // text content, so `pl-*` on the <summary> pushes the marker too. We wrap
  // children in a span with a left margin to put some air between the chevron
  // and the title text.
  return (
    <summary className="cursor-pointer select-none rounded-md px-3 py-2.5 text-sm font-semibold text-slate-800 hover:bg-slate-50">
      <span className="ml-1.5">{children}</span>
    </summary>
  );
}

function DetailsContent({ children }: { children?: React.ReactNode }) {
  // Details bodies are often raw HTML, not parsed markdown (CommonMark only
  // reparses HTML-block content when a blank line separates it from the
  // opening tag). That means a list like `1. Foo\n2. Bar` arrives as a single
  // text node and we need `whitespace-pre-wrap` to keep each item on its
  // own visible line.
  //
  // The downside of `pre-wrap` is that stray leading/trailing newlines
  // between `</summary>` and the body render as blank lines (big top gap,
  // thin bottom gap). We strip those explicitly so the preserved interior
  // newlines still show but the edges sit flush against the padding.
  const normalized = Children.toArray(children);

  while (normalized.length && typeof normalized[0] === "string" && normalized[0].trim() === "") {
    normalized.shift();
  }
  while (
    normalized.length &&
    typeof normalized[normalized.length - 1] === "string" &&
    (normalized[normalized.length - 1] as string).trim() === ""
  ) {
    normalized.pop();
  }
  if (normalized.length) {
    const first = normalized[0];
    if (typeof first === "string") {
      normalized[0] = first.replace(/^\s+/, "");
    }
    const last = normalized[normalized.length - 1];
    if (typeof last === "string") {
      normalized[normalized.length - 1] = last.replace(/\s+$/, "");
    }
  }

  return (
    <div className="whitespace-pre-wrap px-3 py-3 text-sm text-gray-700 [&_p]:my-1 [&_pre]:my-2 [&>*:first-child]:mt-0 [&>*:last-child]:mb-0">
      {normalized}
    </div>
  );
}

export interface NodeChipIcon {
  /** Image URL (e.g. an integration logo). Takes precedence over iconSlug. */
  iconSrc?: string;
  /** Lucide icon slug (kebab-case) used when no image URL is available. */
  iconSlug?: string;
}

export interface NodeChipConfigField {
  label: string;
  value: string;
}

export type NodeChipKind = "trigger" | "component" | "blueprint" | "widget";

export interface NodeChipDetails {
  /** Display name (node.name). */
  name: string;
  /** Node kind — drives chip + hover-card colors (triggers render purple). */
  kind?: NodeChipKind;
  /** Node kind label — "Trigger", "Component", "Blueprint", "Widget". */
  kindLabel?: string;
  /** Human-readable type label from the component/trigger definition (e.g. "HTTP probe"). */
  typeLabel?: string;
  /** Integration label for bound components/triggers (e.g. "Semaphore"). */
  integrationLabel?: string;
  /** Optional icon for the integration (image URL). */
  integrationIcon?: string;
  /** Up to a handful of configuration key/value pairs shown as a compact table. */
  config?: NodeChipConfigField[];
  /** True when the node is paused. Rendered as a small badge in the hover card. */
  paused?: boolean;
  /** True when the node has an error. Rendered as a small badge in the hover card. */
  hasError?: boolean;
}

export interface TriggerTemplateInfo {
  /** Display name of the template (used as the chip label). */
  name: string;
  /** JSON payload preview shown in the hover card and pre-filled into EmitEventModal. */
  payload: unknown;
  /** Available output channels on the trigger; the modal defaults to the first one. */
  outputChannels?: string[];
}

export interface NodeStatusInfo {
  /** Status string from the canvas-runs helpers (e.g. "running", "failed", "approved", or "none" when no executions). */
  status: string;
  /** Tailwind background class for the pill (e.g. "bg-blue-500"). Falls back to the default palette if omitted. */
  badgeColor?: string;
  /** Display label for the pill (uppercased rendering happens in CSS). */
  label?: string;
}

export interface NodeChipContext {
  /** Known node slug -> display name. Slugs not in this map render as "unknown node" chips. */
  nodes?: Record<string, string>;
  /** Known node slug -> icon (image URL or Lucide slug). When present, replaces the leading `@`. */
  icons?: Record<string, NodeChipIcon>;
  /** Known node slug -> rich details used to render a hover-card preview. */
  details?: Record<string, NodeChipDetails>;
  /** Link target for known slugs. Called with the slug. Defaults to `?node=<slug>` on the current page. */
  linkFor?: (slug: string) => string;
  /** Optional click handler instead of navigation (e.g. to open a side panel). */
  onNodeClick?: (slug: string) => void;
  /** Manual Run trigger templates indexed by node slug, then by template slug (kebab-case template name). */
  triggerTemplates?: Record<string, Record<string, TriggerTemplateInfo>>;
  /** Click handler for `@trigger:run/template` chips. When undefined, chips render disabled. */
  onTriggerTemplateRun?: (input: { nodeSlug: string; templateSlug: string }) => void;
  /**
   * Latest execution status per node slug, used by `@node:status` /
   * `[[status:node]]` chips. Slugs not present in the map render as a
   * dashed-grey "unknown status" pill (e.g. for triggers).
   */
  nodeStatuses?: Record<string, NodeStatusInfo | undefined>;
  /** Node slug -> node ID. Used by widget-block row actions to fire events. */
  nodeIds?: Record<string, string>;
  /**
   * Direct event-emit callback used by widget-block row actions. The caller is
   * responsible for resolving `nodeSlug` to a node ID and posting the event.
   * When undefined, action buttons render disabled.
   */
  onEmitEvent?: (input: { nodeSlug: string; channel: string; data: unknown }) => Promise<void>;
  /**
   * Execution-scoped action callback used by widget-block executions rows
   * (`kind: approve | cancel | push-through`). The widget resolves the right
   * `executionId` for the row before invoking. When undefined, action buttons
   * render disabled.
   */
  onExecutionAction?: (input: {
    kind: "approve" | "cancel" | "push-through";
    nodeId: string;
    executionId: string;
  }) => Promise<void>;
}

interface ChipTheme {
  chip: string;
  atSign: string;
  lucide: string;
  tileBg: string;
  tileBorder: string;
  accent: string;
}

const CHIP_THEMES: Record<NodeChipKind | "default", ChipTheme> = {
  trigger: {
    chip: "border-purple-200 bg-purple-50 text-purple-700 hover:bg-purple-100",
    atSign: "text-purple-400",
    lucide: "text-purple-500",
    tileBg: "bg-purple-50",
    tileBorder: "border-purple-100",
    accent: "text-purple-700",
  },
  component: {
    chip: "border-blue-200 bg-blue-50 text-blue-700 hover:bg-blue-100",
    atSign: "text-blue-400",
    lucide: "text-blue-500",
    tileBg: "bg-blue-50",
    tileBorder: "border-blue-100",
    accent: "text-blue-700",
  },
  blueprint: {
    chip: "border-blue-200 bg-blue-50 text-blue-700 hover:bg-blue-100",
    atSign: "text-blue-400",
    lucide: "text-blue-500",
    tileBg: "bg-blue-50",
    tileBorder: "border-blue-100",
    accent: "text-blue-700",
  },
  widget: {
    chip: "border-blue-200 bg-blue-50 text-blue-700 hover:bg-blue-100",
    atSign: "text-blue-400",
    lucide: "text-blue-500",
    tileBg: "bg-blue-50",
    tileBorder: "border-blue-100",
    accent: "text-blue-700",
  },
  default: {
    chip: "border-blue-200 bg-blue-50 text-blue-700 hover:bg-blue-100",
    atSign: "text-blue-400",
    lucide: "text-blue-500",
    tileBg: "bg-blue-50",
    tileBorder: "border-blue-100",
    accent: "text-blue-700",
  },
};

function themeFor(kind?: NodeChipKind): ChipTheme {
  return CHIP_THEMES[kind ?? "default"];
}

function NodeChipLeading({ icon, theme }: { icon?: NodeChipIcon; theme: ChipTheme }) {
  if (icon?.iconSrc) {
    return <img src={icon.iconSrc} alt="" aria-hidden="true" className="h-3 w-3 shrink-0 object-contain" />;
  }
  if (icon?.iconSlug) {
    const Icon = resolveIcon(icon.iconSlug);
    return <Icon className={`h-3 w-3 shrink-0 ${theme.lucide}`} aria-hidden="true" />;
  }
  return <span className={theme.atSign}>@</span>;
}

function KnownNodeChip({
  slug,
  name,
  href,
  onClick,
  icon,
  details,
}: {
  slug: string;
  name: string;
  href?: string;
  onClick?: () => void;
  icon?: NodeChipIcon;
  details?: NodeChipDetails;
}) {
  const label = name || slug;
  const theme = themeFor(details?.kind);
  const classes = `sp-node-ref inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium leading-none ${theme.chip}`;

  const chip = onClick ? (
    <button type="button" className={classes} onClick={onClick} title={`Focus node ${slug}`}>
      <NodeChipLeading icon={icon} theme={theme} />
      {label}
    </button>
  ) : (
    <a className={classes} href={href} title={`Focus node ${slug}`}>
      <NodeChipLeading icon={icon} theme={theme} />
      {label}
    </a>
  );

  if (!details) {
    return chip;
  }

  return (
    <HoverCard openDelay={120} closeDelay={80}>
      <HoverCardTrigger asChild>{chip}</HoverCardTrigger>
      <HoverCardContent
        side="top"
        align="start"
        className="w-72 overflow-hidden rounded-lg border border-slate-200 p-0 text-xs shadow-xl"
        sideOffset={6}
      >
        <NodeChipPreview details={details} icon={icon} theme={theme} />
      </HoverCardContent>
    </HoverCard>
  );
}

function NodeChipPreview({
  details,
  icon,
  theme,
}: {
  details: NodeChipDetails;
  icon?: NodeChipIcon;
  theme: ChipTheme;
}) {
  const subtitleParts = [details.kindLabel, details.typeLabel].filter(Boolean) as string[];
  const hasConfig = Boolean(details.config && details.config.length > 0);
  const hasIntegration = Boolean(details.integrationLabel);

  return (
    <div className="flex flex-col bg-white">
      <div className="flex items-start gap-3 px-4 pt-3.5 pb-3">
        <div
          className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border ${theme.tileBorder} ${theme.tileBg}`}
        >
          <NodeChipPreviewIcon icon={icon} theme={theme} />
        </div>
        <div className="min-w-0 flex-1 pt-0.5">
          <div className="flex items-start gap-2">
            <div className="min-w-0 flex-1 truncate text-[13px] font-semibold leading-tight text-slate-900">
              {details.name}
            </div>
            {(details.paused || details.hasError) && (
              <div className="flex shrink-0 items-center gap-1">
                {details.hasError && (
                  <span className="rounded-full border border-red-200 bg-red-50 px-1.5 py-[1px] text-[10px] font-medium leading-none text-red-700">
                    Error
                  </span>
                )}
                {details.paused && (
                  <span className="rounded-full border border-amber-200 bg-amber-50 px-1.5 py-[1px] text-[10px] font-medium leading-none text-amber-700">
                    Paused
                  </span>
                )}
              </div>
            )}
          </div>
          {subtitleParts.length > 0 && (
            <div className="mt-1 truncate text-[11px] font-medium uppercase tracking-wider text-slate-400">
              {subtitleParts.join(" · ")}
            </div>
          )}
        </div>
      </div>

      {hasConfig && (
        <div className="border-t border-slate-100 bg-slate-50/60 px-4 py-2.5">
          <dl className="grid grid-cols-[auto_1fr] items-baseline gap-x-3 gap-y-1.5 text-[11px]">
            {details.config!.map((field) => (
              <React.Fragment key={field.label}>
                <dt className="whitespace-nowrap text-slate-500">{field.label}</dt>
                <dd className="truncate text-right font-mono text-[11px] text-slate-800" title={field.value}>
                  {field.value}
                </dd>
              </React.Fragment>
            ))}
          </dl>
        </div>
      )}

      {hasIntegration && (
        <div className="flex items-center gap-2 border-t border-slate-100 px-4 py-2">
          {details.integrationIcon ? (
            <img
              src={details.integrationIcon}
              alt=""
              aria-hidden="true"
              className="h-3.5 w-3.5 shrink-0 object-contain"
            />
          ) : (
            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-slate-300" aria-hidden="true" />
          )}
          <span className="truncate text-[11px] text-slate-600">
            <span className="text-slate-400">via </span>
            <span className="font-medium text-slate-700">{details.integrationLabel}</span>
          </span>
        </div>
      )}
    </div>
  );
}

function NodeChipPreviewIcon({ icon, theme }: { icon?: NodeChipIcon; theme: ChipTheme }) {
  if (icon?.iconSrc) {
    return <img src={icon.iconSrc} alt="" aria-hidden="true" className="h-5 w-5 object-contain" />;
  }
  if (icon?.iconSlug) {
    const Icon = resolveIcon(icon.iconSlug);
    return <Icon className={`h-5 w-5 ${theme.lucide}`} aria-hidden="true" />;
  }
  return <span className={`text-base font-semibold ${theme.atSign}`}>@</span>;
}

function UnknownNodeChip({ slug }: { slug: string }) {
  return (
    <span
      className="sp-node-ref inline-flex items-center gap-1 rounded-full border border-dashed border-slate-300 bg-slate-50 px-2 py-0.5 text-[11px] font-medium leading-none text-slate-500"
      title={`Unknown node: ${slug}`}
    >
      <CircleHelp className="h-3 w-3" />@{slug}
    </span>
  );
}

const TRIGGER_RUN_CHIP_CLASSES =
  "sp-trigger-run inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium leading-none border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 focus:outline-none focus:ring-2 focus:ring-emerald-300";

const TRIGGER_RUN_CHIP_DISABLED_CLASSES =
  "sp-trigger-run inline-flex items-center gap-1 rounded-full border border-dashed border-slate-300 bg-slate-50 px-2 py-0.5 text-[11px] font-medium leading-none text-slate-500";

const MAX_PAYLOAD_PREVIEW_CHARS = 800;

function payloadPreview(payload: unknown): string {
  let text: string;
  try {
    text = JSON.stringify(payload ?? {}, null, 2);
  } catch {
    text = String(payload ?? "");
  }
  if (text.length > MAX_PAYLOAD_PREVIEW_CHARS) {
    return `${text.slice(0, MAX_PAYLOAD_PREVIEW_CHARS)}\n…`;
  }
  return text;
}

function TriggerRunChip({
  nodeSlug,
  templateSlug,
  template,
  triggerName,
  onRun,
}: {
  nodeSlug: string;
  templateSlug: string;
  template?: TriggerTemplateInfo;
  triggerName?: string;
  onRun?: () => void;
}) {
  // Unknown trigger or template -> disabled fallback variant.
  if (!template) {
    return (
      <span
        className={TRIGGER_RUN_CHIP_DISABLED_CLASSES}
        title={`Unknown trigger or template: ${nodeSlug}:run/${templateSlug}`}
      >
        <CircleHelp className="h-3 w-3" />
        {nodeSlug}:run/{templateSlug}
      </span>
    );
  }

  const label = template.name || templateSlug;

  // Read-only / no-callback variant.
  if (!onRun) {
    return (
      <span
        className={TRIGGER_RUN_CHIP_DISABLED_CLASSES}
        title={`Run not available here: ${nodeSlug}:run/${templateSlug}`}
      >
        <Play className="h-3 w-3" />
        {label}
      </span>
    );
  }

  const button = (
    <button
      type="button"
      className={TRIGGER_RUN_CHIP_CLASSES}
      onClick={onRun}
      title={`Run ${label} on ${triggerName ?? nodeSlug}`}
      data-trigger={nodeSlug}
      data-template={templateSlug}
    >
      <Play className="h-3 w-3" />
      {label}
    </button>
  );

  return (
    <HoverCard openDelay={120} closeDelay={80}>
      <HoverCardTrigger asChild>{button}</HoverCardTrigger>
      <HoverCardContent
        side="top"
        align="start"
        className="w-80 overflow-hidden rounded-lg border border-slate-200 p-0 text-xs shadow-xl"
        sideOffset={6}
      >
        <div className="flex flex-col bg-white">
          <div className="flex items-start gap-3 px-4 pt-3.5 pb-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-emerald-100 bg-emerald-50">
              <Play className="h-5 w-5 text-emerald-500" aria-hidden="true" />
            </div>
            <div className="min-w-0 flex-1 pt-0.5">
              <div className="min-w-0 truncate text-[13px] font-semibold leading-tight text-slate-900">{label}</div>
              <div className="mt-1 truncate text-[11px] font-medium uppercase tracking-wider text-slate-400">
                Run on {triggerName ?? nodeSlug}
              </div>
            </div>
          </div>
          <div className="border-t border-slate-100 bg-slate-50/60 px-4 py-2.5">
            <div className="mb-1 text-[10px] font-medium uppercase tracking-wider text-slate-400">Payload</div>
            <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded border border-slate-200 bg-white p-2 font-mono text-[11px] text-slate-700">
              {payloadPreview(template.payload)}
            </pre>
          </div>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

function defaultLinkFor(slug: string): string {
  try {
    const url = new URL(window.location.href);
    url.searchParams.set("node", slug);
    // Strip trailing sub-path segments like /readme so the chip lands on the
    // canvas surface itself, not on whatever sibling route we're currently on.
    url.pathname = url.pathname.replace(/\/(readme|settings)\/?$/, "/");
    return `${url.pathname}${url.search}`;
  } catch {
    return `?node=${encodeURIComponent(slug)}`;
  }
}

// Shared classnames for `@node:status` pills. Mirrors the runs-console
// StatusBadge component so the chip is visually identical to what users see
// on canvas / runs rows.
const NODE_STATUS_CHIP_CLASSES =
  "sp-node-status shrink-0 inline-flex items-center justify-center uppercase text-[10px] py-[1.5px] px-[5px] font-semibold rounded tracking-wide text-white";

const NODE_STATUS_DISABLED_CLASSES =
  "sp-node-status inline-flex items-center justify-center rounded border border-dashed border-slate-300 bg-slate-50 px-2 py-0.5 text-[10px] font-medium uppercase leading-none tracking-wide text-slate-500";

function NodeStatusChip({ slug, status }: { slug: string; status?: NodeStatusInfo }) {
  if (!status) {
    return (
      <span className={NODE_STATUS_DISABLED_CLASSES} title={`No status available for ${slug}`}>
        {slug}:status
      </span>
    );
  }

  const badgeColor = status.badgeColor || "bg-gray-400";
  const label = status.label || status.status || "unknown";
  return (
    <span className={`${NODE_STATUS_CHIP_CLASSES} ${badgeColor}`} title={`${slug}: ${label}`}>
      {label}
    </span>
  );
}

function buildSpanComponent(context: NodeChipContext) {
  return function Span(props: React.HTMLAttributes<HTMLSpanElement> & Record<string, unknown>) {
    const className = typeof props.className === "string" ? props.className : "";

    if (className.includes(NODE_STATUS_CLASS)) {
      const nodeAttr = (props["data-node"] as string | undefined) ?? (props.dataNode as string | undefined) ?? "";
      let slug = String(nodeAttr).trim();
      if (!slug) {
        // Fallback: parse `<slug>:status` from the text node content.
        const text = extractTextFromChildren(props.children).trim();
        const m = text.match(/^([a-zA-Z0-9][a-zA-Z0-9_-]*):status$/);
        if (m) slug = m[1];
      }
      if (!slug) return <span {...props} />;

      const status = context.nodeStatuses?.[slug];
      return <NodeStatusChip slug={slug} status={status} />;
    }

    if (className.includes(TRIGGER_RUN_CLASS)) {
      // rehype/react-markdown converts data-* attributes into camelCase props, but
      // since we render via rehype-raw and rehype-sanitize they often stay as
      // `data-trigger`/`data-template`. Be tolerant of both forms.
      const triggerAttr =
        (props["data-trigger"] as string | undefined) ?? (props.dataTrigger as string | undefined) ?? "";
      const templateAttr =
        (props["data-template"] as string | undefined) ?? (props.dataTemplate as string | undefined) ?? "";
      let nodeSlug = String(triggerAttr).trim();
      let templateSlug = String(templateAttr).trim();
      if (!nodeSlug || !templateSlug) {
        // Fallback: parse from text content `<trigger>:run/<template>`.
        const text = extractTextFromChildren(props.children).trim();
        const m = text.match(/^([a-zA-Z0-9][a-zA-Z0-9_-]*):run\/([a-zA-Z0-9][a-zA-Z0-9_-]*)$/);
        if (m) {
          nodeSlug = nodeSlug || m[1];
          templateSlug = templateSlug || m[2];
        }
      }
      if (!nodeSlug || !templateSlug) {
        return <span {...props} />;
      }

      const template = context.triggerTemplates?.[nodeSlug]?.[templateSlug];
      const triggerName = context.nodes?.[nodeSlug];
      const onRun = context.onTriggerTemplateRun
        ? () => context.onTriggerTemplateRun?.({ nodeSlug, templateSlug })
        : undefined;

      return (
        <TriggerRunChip
          nodeSlug={nodeSlug}
          templateSlug={templateSlug}
          template={template}
          triggerName={triggerName}
          onRun={onRun}
        />
      );
    }

    if (!className.includes(NODE_REF_CLASS)) {
      return <span {...props} />;
    }

    const slug = extractTextFromChildren(props.children).trim();
    if (!slug) return <span {...props} />;

    const known = context.nodes ?? {};
    const isKnown = Object.prototype.hasOwnProperty.call(known, slug);
    if (!isKnown) {
      return <UnknownNodeChip slug={slug} />;
    }

    const name = known[slug] ?? slug;
    const icon = context.icons?.[slug];
    const details = context.details?.[slug];
    const linkFor = context.linkFor ?? defaultLinkFor;

    if (context.onNodeClick) {
      return (
        <KnownNodeChip
          slug={slug}
          name={name}
          icon={icon}
          details={details}
          onClick={() => context.onNodeClick?.(slug)}
        />
      );
    }

    return <KnownNodeChip slug={slug} name={name} icon={icon} details={details} href={linkFor(slug)} />;
  };
}

//
// Default typography for full-page readme-style rendering: a proper heading
// scale, paragraph spacing, and list indentation. Compact surfaces (e.g. the
// Reports panel) pass their own `className` to opt out of this scale.
//
const DEFAULT_TYPOGRAPHY_CLASS = [
  "text-sm text-gray-800 leading-relaxed",
  "[&>:first-child]:mt-0",
  "[&_p]:my-2",
  "[&_ul]:my-2 [&_ul]:pl-5 [&_ul]:list-disc",
  "[&_ol]:my-2 [&_ol]:pl-5 [&_ol]:list-decimal",
  "[&_li]:my-0.5",
  "[&_h1]:text-2xl [&_h1]:font-semibold [&_h1]:tracking-tight [&_h1]:mt-6 [&_h1]:mb-3",
  "[&_h2]:text-xl [&_h2]:font-semibold [&_h2]:mt-5 [&_h2]:mb-2 [&_h2]:border-b [&_h2]:border-slate-200 [&_h2]:pb-1",
  "[&_h3]:text-lg [&_h3]:font-semibold [&_h3]:mt-4 [&_h3]:mb-2",
  "[&_h4]:text-base [&_h4]:font-semibold [&_h4]:mt-3 [&_h4]:mb-1",
  "[&_h5]:text-sm [&_h5]:font-semibold [&_h5]:mt-2 [&_h5]:mb-1",
  "[&_h6]:text-xs [&_h6]:font-semibold [&_h6]:uppercase [&_h6]:tracking-wide [&_h6]:text-gray-600 [&_h6]:mt-2 [&_h6]:mb-1",
].join(" ");

interface CanvasMarkdownProps {
  children: string;
  className?: string;
  nodeRefs?: NodeChipContext;
  canvasId?: string;
}

export function CanvasMarkdown({ children, className, nodeRefs, canvasId }: CanvasMarkdownProps) {
  const components = React.useMemo(
    () => ({
      code: buildInlineCodeComponent({ canvasId, nodeRefs }),
      blockquote: Blockquote,
      a: Anchor,
      table: Table,
      th: Th,
      td: Td,
      img: Img,
      hr: Hr,
      details: Details,
      summary: Summary,
      span: buildSpanComponent(nodeRefs ?? {}),
    }),
    [nodeRefs, canvasId],
  );

  const wrapperClassName = className ?? DEFAULT_TYPOGRAPHY_CLASS;

  return (
    <div className={wrapperClassName}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkNodeRefs]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, sanitizeSchema], rehypeHighlight]}
        components={components as never}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
