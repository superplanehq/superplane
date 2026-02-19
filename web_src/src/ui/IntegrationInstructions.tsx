import ReactMarkdown from "react-markdown";
import { ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";

const INSTRUCTIONS_CLASSES =
  "rounded-md border border-orange-950/15 bg-orange-100 p-4 text-sm text-gray-800 dark:border-blue-800 dark:bg-blue-950/30 dark:text-gray-200 [&_a]:!underline [&_a]:underline-offset-2 [&_a]:decoration-2 [&_a]:decoration-current [&_ol]:list-decimal [&_ol]:ml-5 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-5 [&_ul]:space-y-1";

export interface IntegrationInstructionsProps {
  /** Markdown description (e.g. setup steps) */
  description?: string | null;
  /** When provided, a "Continue" button is shown that calls this (e.g. open OAuth URL) */
  onContinue?: () => void;
  /** Optional class name for the wrapper */
  className?: string;
}

/**
 * Shared block for integration setup/configuration instructions.
 * Same styling everywhere: bg-blue-50, border-blue-200, text-gray-800.
 * Used in sidebar (Create/Configure integration dialogs) and org/integrations.
 */
export function IntegrationInstructions({ description, onContinue, className = "" }: IntegrationInstructionsProps) {
  if (!description?.trim()) return null;

  const normalizedDescription = description.replace(/\r\n/g, "\n").replace(/\n(?!\n)/g, "  \n");

  return (
    <div className={`${INSTRUCTIONS_CLASSES} ${className}`.trim()}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
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
              code: ({ children }) => <code className="rounded bg-black/10 px-1 text-xs">{children}</code>,
              strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
              em: ({ children }) => <em className="italic">{children}</em>,
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
