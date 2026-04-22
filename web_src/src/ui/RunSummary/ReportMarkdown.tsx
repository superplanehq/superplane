import React, { Children, isValidElement } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import { Info, AlertTriangle, OctagonAlert, Lightbulb, MessageCircleWarning, ExternalLink } from "lucide-react";

import "highlight.js/styles/github.css";

//
// Allow <details>/<summary> (collapsible sections) to survive sanitization,
// on top of the default-safe rehype-sanitize schema.
//
const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary"],
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

const components = {
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
};

interface ReportMarkdownProps {
  children: string;
  className?: string;
}

export function ReportMarkdown({ children, className }: ReportMarkdownProps) {
  return (
    <div className={className}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, sanitizeSchema], rehypeHighlight]}
        components={components as never}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
