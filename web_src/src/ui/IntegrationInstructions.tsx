import { isValidElement, type ComponentProps, type ReactNode, useState } from "react";
import ReactMarkdown from "react-markdown";
import { Check, Copy, ExternalLink, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

const INSTRUCTIONS_CLASSES =
  "rounded-md border border-orange-950/15 bg-orange-100 p-4 text-sm text-gray-800 dark:border-orange-900/40 dark:bg-orange-950/30 dark:text-gray-200 [&_a]:!underline [&_a]:underline-offset-2 [&_a]:decoration-2 [&_a]:decoration-current [&_ol]:list-decimal [&_ol]:ml-5 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-5 [&_ul]:space-y-1";

type MarkdownCodeProps = ComponentProps<"code"> & {
  inline?: boolean;
};

async function copyTextToClipboard(text: string): Promise<boolean> {
  try {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch (_err) {}

  try {
    if (typeof document === "undefined") return false;

    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.top = "-9999px";
    textarea.style.left = "-9999px";
    document.body.appendChild(textarea);
    textarea.select();

    const copied = document.execCommand("copy");
    document.body.removeChild(textarea);
    return copied;
  } catch (_err) {
    return false;
  }
}

function extractTextFromNode(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }

  if (Array.isArray(node)) {
    return node.map((child) => extractTextFromNode(child)).join("");
  }

  if (isValidElement<{ children?: ReactNode }>(node)) {
    return extractTextFromNode(node.props.children);
  }

  return "";
}

function CopyableMarkdownPre({ children }: { children: ReactNode }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    const content = extractTextFromNode(children).replace(/\n$/, "");
    if (!content) return;

    const didCopy = await copyTextToClipboard(content);
    if (!didCopy) return;

    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="mb-2 flex w-full min-w-0 items-start gap-2">
      <pre className="w-0 min-w-0 flex-1 overflow-x-auto overflow-y-hidden rounded bg-black/10 p-2 text-xs">
        {children}
      </pre>
      <button
        type="button"
        onClick={handleCopy}
        className="mt-1 shrink-0 rounded bg-black/10 p-1"
        title={copied ? "Copied!" : "Copy code"}
        aria-label={copied ? "Copied!" : "Copy code"}
      >
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
    </div>
  );
}

function CopyableInlineCode({ children }: { children: ReactNode }) {
  const [copied, setCopied] = useState(false);
  const content = extractTextFromNode(children);
  const shouldUseBlockStyle = /^(https?:\/\/|arn:)/i.test(content) || content.length >= 32;

  const handleCopy = async () => {
    if (!content) return;

    const didCopy = await copyTextToClipboard(content);
    if (!didCopy) return;

    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return shouldUseBlockStyle ? (
    <span className="inline-flex max-w-full items-center gap-1 align-middle">
      <code
        role="button"
        tabIndex={0}
        onClick={handleCopy}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            void handleCopy();
          }
        }}
        className="inline-block max-w-[28rem] cursor-copy overflow-x-auto overflow-y-hidden rounded bg-black/10 px-1.5 py-0.5 font-mono text-[11px] leading-4 whitespace-nowrap"
        title={copied ? "Copied!" : "Click to copy"}
        aria-label={copied ? "Copied!" : "Click to copy"}
      >
        {children}
      </code>
      <button
        type="button"
        onClick={handleCopy}
        className="shrink-0 rounded bg-black/10 p-1"
        title={copied ? "Copied!" : "Copy code"}
        aria-label={copied ? "Copied!" : "Copy code"}
      >
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
    </span>
  ) : (
    <button
      type="button"
      onClick={handleCopy}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          void handleCopy();
        }
      }}
      className="group inline-flex max-w-full items-center gap-1 rounded border border-black/20 bg-black/5 px-1.5 py-0.5 align-middle font-mono text-[11px] leading-4 text-gray-900 transition-colors hover:bg-black/10 focus:outline-none focus:ring-1 focus:ring-black/25 dark:border-white/20 dark:bg-white/10 dark:text-gray-100 dark:hover:bg-white/15"
      title={copied ? "Copied!" : "Click to copy"}
      aria-label={copied ? "Copied!" : "Click to copy"}
    >
      <span className="min-w-0 break-all text-left">{children}</span>
      {copied ? <Check className="h-3 w-3 shrink-0" /> : <Copy className="h-3 w-3 shrink-0 opacity-70" />}
    </button>
  );
}

export interface IntegrationInstructionsProps {
  /** Markdown description (e.g. setup steps) */
  description?: string | null;
  /** Optional actions rendered as buttons below the instruction text */
  actions?: Array<{
    label: string;
    onClick: () => void;
    external?: boolean;
    disabled?: boolean;
    isPending?: boolean;
  }>;
  /** Optional class name for the wrapper */
  className?: string;
}

/**
 * Shared block for integration setup/configuration instructions.
 * Used in sidebar (Create/Configure integration dialogs) and org/integrations.
 */
export function IntegrationInstructions({ description, actions, className = "" }: IntegrationInstructionsProps) {
  if (!description?.trim()) return null;

  const normalizedDescription = description.replace(/\r\n/g, "\n").replace(/\n(?!\n)/g, "  \n");

  return (
    <div className={`min-w-0 ${INSTRUCTIONS_CLASSES} ${className}`.trim()}>
      <div className="space-y-3">
        <div className="min-w-0">
          <ReactMarkdown
            components={{
              h1: ({ children }) => <h1 className="text-base font-semibold mt-2 mb-2">{children}</h1>,
              h2: ({ children }) => <h2 className="text-base font-semibold mt-2 mb-2">{children}</h2>,
              h3: ({ children }) => <h3 className="text-sm font-semibold mt-2 mb-1">{children}</h3>,
              h4: ({ children }) => <h4 className="text-sm font-medium mt-2 mb-1">{children}</h4>,
              p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
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
              pre: ({ children }) => <CopyableMarkdownPre>{children}</CopyableMarkdownPre>,
              code: ({ children, className }: MarkdownCodeProps) => {
                const content = extractTextFromNode(children);
                const isBlockCode = Boolean(className?.includes("language-")) || content.includes("\n");

                if (isBlockCode) {
                  return (
                    <code className={className ? `${className} whitespace-pre` : "whitespace-pre"}>{children}</code>
                  );
                }

                return <CopyableInlineCode>{children}</CopyableInlineCode>;
              },
              strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
              em: ({ children }) => <em className="italic">{children}</em>,
            }}
          >
            {normalizedDescription}
          </ReactMarkdown>
        </div>
        {actions && actions.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {actions.map((action, index) => (
              <Button
                key={`${action.label}-${index}`}
                type="button"
                variant="outline"
                onClick={action.onClick}
                className="px-3 py-1.5"
                disabled={action.disabled || action.isPending}
              >
                {action.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                {!action.isPending && action.external ? <ExternalLink className="w-4 h-4" /> : null}
                {action.label}
              </Button>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
