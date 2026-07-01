import React from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/ui/CopyButton";

const INSTRUCTIONS_V2_CLASSES =
  "text-sm text-gray-800 dark:text-gray-200 [&_a]:!underline [&_a]:underline-offset-2 [&_a]:decoration-2 [&_a]:decoration-current [&_ol]:list-decimal [&_ol]:ml-5 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-5 [&_ul]:space-y-1";

/** Matches horizontal rules inside markdown; reuse for external separators that should align with them. */
export const INTEGRATION_INSTRUCTIONS_HR_CLASS = "my-4 border-0 border-t border-gray-300 dark:border-gray-600";

/** Subtle scrollbar for markdown table overflow (Firefox + WebKit); horizontal bar uses height. */
const MARKDOWN_TABLE_SCROLL_CLASSES =
  "[scrollbar-width:thin] [scrollbar-color:rgb(156_163_175)_rgb(243_244_246)] dark:[scrollbar-color:rgb(107_114_128)_rgb(31_41_55)] [&::-webkit-scrollbar]:h-2 [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-gray-400/85 dark:[&::-webkit-scrollbar-thumb]:bg-gray-500/85 [&::-webkit-scrollbar-track]:rounded-full [&::-webkit-scrollbar-track]:bg-gray-100 dark:[&::-webkit-scrollbar-track]:bg-gray-800/90";

export interface InstructionsProps {
  description?: string | null;
  onContinue?: () => void;
  className?: string;
}

function extractTextFromNode(node: React.ReactNode): string {
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }

  if (Array.isArray(node)) {
    return node.map(extractTextFromNode).join("");
  }

  if (React.isValidElement<{ children?: React.ReactNode }>(node)) {
    return extractTextFromNode(node.props.children);
  }

  return "";
}

function extractCodeBlock(children: React.ReactNode): { code: string; language?: string } {
  const childArray = React.Children.toArray(children);

  const codeElement = childArray.find(
    (
      child,
    ): child is React.ReactElement<{
      className?: string;
      children?: React.ReactNode;
    }> => React.isValidElement(child) && child.type === "code",
  );

  if (!codeElement) {
    return { code: extractTextFromNode(children).replace(/\n$/, "") };
  }

  const className = codeElement.props.className;
  const language = className?.startsWith("language-") ? className.slice("language-".length) : undefined;

  return {
    code: extractTextFromNode(codeElement.props.children).replace(/\n$/, ""),
    language,
  };
}

export function Instructions({ description, onContinue, className = "" }: InstructionsProps) {
  if (!description?.trim()) return null;

  const normalizedDescription = description.replace(/\r\n/g, "\n");

  return (
    <div className={`${INSTRUCTIONS_V2_CLASSES} ${className}`.trim()}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <ReactMarkdown
            remarkPlugins={[remarkGfm, remarkBreaks]}
            components={{
              h1: ({ children }) => <h1 className="text-base font-semibold mt-4 mb-4">{children}</h1>,
              h2: ({ children }) => <h2 className="text-base font-semibold mt-3 mb-3">{children}</h2>,
              h3: ({ children }) => <h3 className="text-sm font-semibold mt-2 mb-1">{children}</h3>,
              h4: ({ children }) => <h4 className="text-sm font-medium mt-2 mb-1">{children}</h4>,
              p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
              pre: ({ children }) => {
                const { code } = extractCodeBlock(children);
                return (
                  <div className="relative my-3 overflow-hidden rounded-md border border-black/15 bg-gray-800/80 text-gray-100 dark:border-white/20 dark:bg-gray-800/90">
                    <div className="absolute right-2 top-2 z-10">
                      <CopyButton text={code} dark />
                    </div>
                    <pre className="overflow-x-auto px-4 py-3 pr-12 text-xs leading-relaxed">{children}</pre>
                  </div>
                );
              },
              blockquote: ({ children }) => (
                <blockquote className="mb-2 rounded-md border border-gray-300 bg-gray-50 p-3 text-sm last:mb-0 dark:border-gray-700 dark:bg-gray-900/60">
                  {children}
                </blockquote>
              ),
              hr: () => <hr className={INTEGRATION_INSTRUCTIONS_HR_CLASS} />,
              ul: ({ children }) => <ul className="list-disc ml-5 space-y-1 mb-2">{children}</ul>,
              ol: ({ children }) => <ol className="list-decimal ml-5 space-y-1 mb-2">{children}</ol>,
              li: ({ children }) => <li>{children}</li>,
              a: ({ children, href }) => (
                <a
                  className="!underline underline-offset-2 decoration-2 decoration-current"
                  target="_blank"
                  rel="noopener noreferrer"
                  href={href}
                >
                  {children}
                </a>
              ),
              code: ({ children, className: codeClassName }) => {
                const isBlockCode = Boolean(codeClassName?.includes("language-"));
                if (isBlockCode) {
                  return <code className={codeClassName}>{children}</code>;
                }

                const inlineText = extractTextFromNode(children).trim();
                const codeEl = <code className="rounded bg-black/10 px-1.5 py-0.5 font-mono text-xs">{children}</code>;

                if (!inlineText) {
                  return codeEl;
                }

                return (
                  <span className="inline-flex max-w-full items-center gap-0.5 align-middle">
                    {codeEl}
                    <CopyButton text={inlineText} />
                  </span>
                );
              },
              strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
              em: ({ children }) => <em className="italic">{children}</em>,
              table: ({ children }) => (
                <div
                  className={`my-3 overflow-x-auto rounded-md border border-gray-300 bg-white dark:border-gray-700 dark:bg-gray-900/50 ${MARKDOWN_TABLE_SCROLL_CLASSES}`}
                >
                  <table className="w-full min-w-max border-collapse text-left text-sm">{children}</table>
                </div>
              ),
              thead: ({ children }) => (
                <thead className="border-b border-gray-200 dark:border-gray-700">{children}</thead>
              ),
              tbody: ({ children }) => (
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">{children}</tbody>
              ),
              tr: ({ children }) => <tr>{children}</tr>,
              th: ({ children }) => (
                <th className="whitespace-nowrap bg-gray-50 px-3 py-2.5 font-semibold text-gray-900 dark:bg-gray-800/80 dark:text-gray-100">
                  {children}
                </th>
              ),
              td: ({ children }) => (
                <td className="px-3 py-2.5 align-top text-gray-800 dark:text-gray-200">{children}</td>
              ),
            }}
          >
            {normalizedDescription}
          </ReactMarkdown>
        </div>
        {onContinue && (
          <Button type="button" variant="outline" onClick={onContinue} className="shrink-0 px-3 py-1.5">
            <ExternalLink className="w-4 h-4" />
            Continue
          </Button>
        )}
      </div>
    </div>
  );
}
