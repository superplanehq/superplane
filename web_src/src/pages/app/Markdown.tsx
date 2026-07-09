import { Children, isValidElement } from "react";
import type { ComponentProps, ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import { defaultUrlTransform } from "react-markdown";
import type { ExtraProps } from "react-markdown";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";

import { MarkdownCode } from "@/components/AgentSidebar/widgets/MarkdownCode";
import { MermaidWidget } from "@/components/AgentSidebar/widgets/MermaidWidget";
import { NodeChipFromLink } from "@/components/AgentSidebar/widgets/NodeChip";
import { cn } from "@/lib/utils";

import { markdownHeadingClassName } from "./markdownHeadingStyles";
import {
  MARKDOWN_TABLE_CLASSES,
  MARKDOWN_TABLE_DATA_CLASSES,
  MARKDOWN_TABLE_EMPHASIS_CLASSES,
  MARKDOWN_TABLE_HEAD_CLASSES,
} from "./markdownTableStyles";

/**
 * Tailwind class string shared by every full-document markdown renderer in the
 * app. We deliberately do not use the official `prose` plugin so headings,
 * code blocks, tables, and `<details>` stay visually consistent with the
 * canvas chrome at small panel sizes.
 *
 * Heading spacing is applied on the rendered `<h1>`–`<h4>` elements (see
 * `markdownHeadingComponents`) because parent `[&_h2]:mt-*` utilities are not
 * reliably generated/applied in our Tailwind v4 setup.
 */
const MARKDOWN_CONTENT_CLASSES =
  "max-w-none text-[13px] text-slate-800 dark:text-gray-100 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed " +
  "[&_strong]:font-semibold [&_b]:font-semibold " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal " +
  "[&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 dark:[&_blockquote]:border-gray-600 " +
  "[&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs dark:[&_code]:bg-gray-800 " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 dark:[&_pre]:bg-gray-800 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current " +
  "[&_details]:my-3 [&_details]:rounded-md [&_details]:border [&_details]:border-slate-200 [&_details]:bg-slate-50/60 [&_details]:px-3 [&_details]:py-2 dark:[&_details]:border-gray-700 dark:[&_details]:bg-gray-800/60 " +
  "[&_details>summary]:flex [&_details>summary]:items-center [&_details>summary]:cursor-pointer [&_details>summary]:select-none [&_details>summary]:text-[13px] [&_details>summary]:font-semibold [&_details>summary]:text-slate-900 [&_details>summary]:list-none [&_details>summary]:marker:hidden [&_details>summary]:hover:text-gray-600 dark:[&_details>summary]:text-gray-100 dark:[&_details>summary]:hover:text-gray-400 " +
  "[&_details>summary]:before:content-['▸'] [&_details>summary]:before:mr-2 [&_details>summary]:before:text-slate-500 [&_details>summary]:before:transition-transform [&_details>summary]:before:duration-200 dark:[&_details>summary]:before:text-gray-400 " +
  "[&_details[open]>summary]:mb-3 [&_details[open]>summary]:before:rotate-90 " +
  "[&_details>*:last-child]:mb-0";

/**
 * Sanitize schema extending the rehype-sanitize defaults with `<details>` /
 * `<summary>` (plus the `open` attribute) so collapsible sections can be
 * authored directly in markdown without weakening the rest of the policy
 * around scripts, event handlers, and inline styles.
 */
const MARKDOWN_SANITIZE_SCHEMA = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary"],
  attributes: {
    ...(defaultSchema.attributes ?? {}),
    a: [...(defaultSchema.attributes?.a ?? []), "title"],
    details: [...(defaultSchema.attributes?.details ?? []), "open"],
  },
  protocols: {
    ...(defaultSchema.protocols ?? {}),
    href: [...(defaultSchema.protocols?.href ?? []), "node"],
  },
};

interface MarkdownContentProps {
  content: string;
  className?: string;
  canvasId?: string;
  organizationId?: string;
  "data-testid"?: string;
}

/**
 * Render a markdown string with the standard GFM + line-break + sanitized-raw
 * HTML pipeline used across the app (console panels, file viewer, etc).
 * Returns `null` when the content is empty (or whitespace-only) so the caller
 * can decide whether to show its own empty state.
 *
 * Only line endings are normalized; leading/trailing whitespace is preserved
 * so file viewers render exactly what's on disk (e.g. an indented code block
 * at the very start of a file stays an indented code block).
 */
export function MarkdownContent({
  content,
  className,
  canvasId,
  organizationId,
  "data-testid": dataTestId,
}: MarkdownContentProps) {
  const normalized = content.replace(/\r\n/g, "\n");
  if (!normalized.trim()) return null;
  return (
    <div className={cn(MARKDOWN_CONTENT_CLASSES, className)} data-testid={dataTestId}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, MARKDOWN_SANITIZE_SCHEMA]]}
        urlTransform={(url) => (isNodeLink(url) ? url : defaultUrlTransform(url))}
        components={{
          h1: ({ children, ...props }) => (
            <h1 className={markdownHeadingClassName("h1")} {...props}>
              {children}
            </h1>
          ),
          h2: ({ children, ...props }) => (
            <h2 className={markdownHeadingClassName("h2")} {...props}>
              {children}
            </h2>
          ),
          h3: ({ children, ...props }) => (
            <h3 className={markdownHeadingClassName("h3")} {...props}>
              {children}
            </h3>
          ),
          h4: ({ children, ...props }) => (
            <h4 className={markdownHeadingClassName("h4")} {...props}>
              {children}
            </h4>
          ),
          table: ({ children, ...props }) => (
            <table className={MARKDOWN_TABLE_CLASSES} {...props}>
              {children}
            </table>
          ),
          th: ({ children, ...props }) => (
            <th className={MARKDOWN_TABLE_HEAD_CLASSES} {...props}>
              {children}
            </th>
          ),
          td: ({ children, ...props }) => (
            <td className={MARKDOWN_TABLE_DATA_CLASSES} {...props}>
              {children}
            </td>
          ),
          strong: ({ children, ...props }) => (
            <strong className={MARKDOWN_TABLE_EMPHASIS_CLASSES} {...props}>
              {children}
            </strong>
          ),
          b: ({ children, ...props }) => (
            <b className={MARKDOWN_TABLE_EMPHASIS_CLASSES} {...props}>
              {children}
            </b>
          ),
          a: ({ children, href, node: _node, ...props }) => (
            <MarkdownLink href={href} canvasId={canvasId} organizationId={organizationId} {...props}>
              {children}
            </MarkdownLink>
          ),
          code: MarkdownCodeWithDiagrams,
          pre: MarkdownPre,
        }}
      >
        {normalized}
      </ReactMarkdown>
    </div>
  );
}

type MarkdownNode = NonNullable<ExtraProps["node"]>;
type MarkdownElementChild = Extract<MarkdownNode["children"][number], { type: "element" }>;

function MarkdownPre({ children, node, ...props }: ComponentProps<"pre"> & ExtraProps) {
  if (hasLanguageCodeNode(node) || hasLanguageCodeChild(children)) {
    return <>{children}</>;
  }

  return <pre {...props}>{children}</pre>;
}

function MarkdownCodeWithDiagrams({
  className,
  children,
  ...props
}: ComponentProps<"code"> & { children?: ReactNode }) {
  const language = /language-(\w+)/.exec(className || "")?.[1];
  const code = String(children).replace(/\n$/, "");

  if (language === "mermaid") {
    return <MermaidWidget content={code} />;
  }

  return (
    <MarkdownCode className={className} {...props}>
      {children}
    </MarkdownCode>
  );
}

function MarkdownLink({
  href,
  children,
  canvasId,
  organizationId,
  ...props
}: ComponentProps<"a"> & { canvasId?: string; organizationId?: string }) {
  const nodeMatch = href?.match(/^node:(.+)$/);
  if (nodeMatch && canvasId && organizationId) {
    const label = typeof children === "string" ? children : undefined;
    return (
      <NodeChipFromLink nodeId={nodeMatch[1]} rawLabel={label} canvasId={canvasId} organizationId={organizationId} />
    );
  }

  return (
    <a href={href} {...props}>
      {children}
    </a>
  );
}

function isNodeLink(url: string): boolean {
  return url.startsWith("node:");
}

function hasLanguageCodeChild(children: ReactNode): boolean {
  const child = Children.toArray(children)[0];
  return isValidElement<{ className?: string }>(child) && /^language-\w+/.test(child.props.className || "");
}

function hasLanguageCodeNode(node?: ExtraProps["node"]): boolean {
  const codeNode = node?.children?.find(
    (child): child is MarkdownElementChild => child.type === "element" && child.tagName === "code",
  );
  return getClassNames(codeNode?.properties?.className).some((className) => /^language-\w+$/.test(className));
}

function getClassNames(className: unknown): string[] {
  if (typeof className === "string") {
    return className.split(/\s+/).filter(Boolean);
  }

  if (Array.isArray(className)) {
    return className.filter((name): name is string => typeof name === "string");
  }

  return [];
}
