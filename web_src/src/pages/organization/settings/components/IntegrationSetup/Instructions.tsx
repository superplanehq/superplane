import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/ui/CopyButton";
import { extractCodeBlock, extractTextFromNode } from "@/lib/markdownCode";

const INSTRUCTIONS_V2_CLASSES =
  "text-sm text-content-primary [&_a]:!underline [&_a]:underline-offset-2 [&_a]:decoration-2 [&_a]:decoration-current [&_ol]:list-decimal [&_ol]:ml-5 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-5 [&_ul]:space-y-1";

/** Matches horizontal rules inside markdown; reuse for external separators that should align with them. */
export const INTEGRATION_INSTRUCTIONS_HR_CLASS = "my-4 border-0 border-t border-edge-default";

/** Subtle scrollbar for markdown table overflow (Firefox + WebKit); horizontal bar uses height. */
const MARKDOWN_TABLE_SCROLL_CLASSES =
  "[scrollbar-width:thin] [scrollbar-color:var(--content-muted)_var(--surface-subtle)] [&::-webkit-scrollbar]:h-2 [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-content-muted/85 [&::-webkit-scrollbar-track]:rounded-full [&::-webkit-scrollbar-track]:bg-surface-subtle";

export interface InstructionsProps {
  description?: string | null;
  onContinue?: () => void;
  className?: string;
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
                  <div className="relative my-3 overflow-hidden rounded-md border border-edge-strong bg-surface-raised text-content-primary">
                    <div className="absolute right-2 top-2 z-10">
                      <CopyButton text={code} dark />
                    </div>
                    <pre className="overflow-x-auto px-4 py-3 pr-12 text-xs leading-relaxed">{children}</pre>
                  </div>
                );
              },
              blockquote: ({ children }) => (
                <blockquote className="mb-2 rounded-md border border-edge-default bg-surface-subtle p-3 text-sm last:mb-0">
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
                const codeEl = (
                  <code className="rounded bg-content-primary/10 px-1.5 py-0.5 font-mono text-xs">{children}</code>
                );

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
                  className={`my-3 overflow-x-auto rounded-md border border-edge-default bg-surface-raised ${MARKDOWN_TABLE_SCROLL_CLASSES}`}
                >
                  <table className="w-full min-w-max border-collapse text-left text-sm">{children}</table>
                </div>
              ),
              thead: ({ children }) => <thead className="border-b border-edge-default">{children}</thead>,
              tbody: ({ children }) => <tbody className="divide-y divide-edge-subtle">{children}</tbody>,
              tr: ({ children }) => <tr>{children}</tr>,
              th: ({ children }) => (
                <th className="whitespace-nowrap bg-surface-subtle px-3 py-2.5 font-semibold text-content-primary">
                  {children}
                </th>
              ),
              td: ({ children }) => <td className="px-3 py-2.5 align-top text-content-primary">{children}</td>,
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
