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
} from "lucide-react";

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

const NODE_TOKEN_RE =
  /(\[\[node:([a-zA-Z0-9][a-zA-Z0-9_-]*)\]\]|(?<![A-Za-z0-9_])@([a-zA-Z][a-zA-Z0-9_-]*))/g;

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
            const slug = match[2] || match[3];
            if (match.index > lastIndex) {
              replacements.push({ type: "text", value: value.slice(lastIndex, match.index) });
            }
            replacements.push({
              type: "html",
              value: `<span class="${NODE_REF_CLASS}">${escapeHtml(slug)}</span>`,
            });
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
    span: [...((defaultSchema.attributes ?? {}).span ?? []), "className", "class"],
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

function InlineCode({ children, className }: { children?: React.ReactNode; className?: string }) {
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

  return <code className={className}>{children}</code>;
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
  return (
    <summary className="cursor-pointer select-none rounded-md px-3 py-2 text-xs font-medium text-gray-600 hover:bg-slate-50">
      {children}
    </summary>
  );
}

function DetailsContent({ children }: { children?: React.ReactNode }) {
  return (
    <div className="whitespace-pre-wrap px-3 py-1.5 text-sm text-gray-700 [&_p]:my-1 [&_pre]:my-2">{children}</div>
  );
}

export interface NodeChipContext {
  /** Known node slug -> display name. Slugs not in this map render as "unknown node" chips. */
  nodes?: Record<string, string>;
  /** Link target for known slugs. Called with the slug. Defaults to `?node=<slug>` on the current page. */
  linkFor?: (slug: string) => string;
  /** Optional click handler instead of navigation (e.g. to open a side panel). */
  onNodeClick?: (slug: string) => void;
}

function KnownNodeChip({
  slug,
  name,
  href,
  onClick,
}: {
  slug: string;
  name: string;
  href?: string;
  onClick?: () => void;
}) {
  const label = name || slug;
  const classes =
    "sp-node-ref inline-flex items-center gap-1 rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-[11px] font-medium leading-none text-blue-700 hover:bg-blue-100";

  if (onClick) {
    return (
      <button type="button" className={classes} onClick={onClick} title={`Focus node ${slug}`}>
        <span className="text-blue-400">@</span>
        {label}
      </button>
    );
  }

  return (
    <a className={classes} href={href} title={`Focus node ${slug}`}>
      <span className="text-blue-400">@</span>
      {label}
    </a>
  );
}

function UnknownNodeChip({ slug }: { slug: string }) {
  return (
    <span
      className="sp-node-ref inline-flex items-center gap-1 rounded-full border border-dashed border-slate-300 bg-slate-50 px-2 py-0.5 text-[11px] font-medium leading-none text-slate-500"
      title={`Unknown node: ${slug}`}
    >
      <CircleHelp className="h-3 w-3" />
      @{slug}
    </span>
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

function buildSpanComponent(context: NodeChipContext) {
  return function Span(props: React.HTMLAttributes<HTMLSpanElement>) {
    const className = typeof props.className === "string" ? props.className : "";
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
    const linkFor = context.linkFor ?? defaultLinkFor;

    if (context.onNodeClick) {
      return <KnownNodeChip slug={slug} name={name} onClick={() => context.onNodeClick?.(slug)} />;
    }

    return <KnownNodeChip slug={slug} name={name} href={linkFor(slug)} />;
  };
}

interface CanvasMarkdownProps {
  children: string;
  className?: string;
  nodeRefs?: NodeChipContext;
}

export function CanvasMarkdown({ children, className, nodeRefs }: CanvasMarkdownProps) {
  const components = React.useMemo(
    () => ({
      code: InlineCode,
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
    [nodeRefs],
  );

  return (
    <div className={className}>
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
