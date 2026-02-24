import ReactMarkdown from "react-markdown";
import { ExternalLink, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

const INSTRUCTIONS_CLASSES =
  "rounded-md border border-orange-950/15 bg-orange-100 p-4 text-sm text-gray-800 dark:border-orange-900/40 dark:bg-orange-950/30 dark:text-gray-200 [&_a]:!underline [&_a]:underline-offset-2 [&_a]:decoration-2 [&_a]:decoration-current [&_ol]:list-decimal [&_ol]:ml-5 [&_ol]:space-y-1 [&_ul]:list-disc [&_ul]:ml-5 [&_ul]:space-y-1";

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
    <div className={`${INSTRUCTIONS_CLASSES} ${className}`.trim()}>
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
              code: ({ children }) => <code className="rounded bg-black/10 px-1 text-xs">{children}</code>,
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
