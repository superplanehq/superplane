import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { cn } from "@/lib/utils";

interface IntegrationButtonProps {
  /** Integration definition name (e.g. "github", "cloudflare", "aws.lambda") */
  integrationName: string;
  /** Display label (from markdown link text) */
  label?: string;
}

/**
 * Renders an integration reference as a clickable button with the vendor icon.
 * Dispatches a CustomEvent so the parent page can open the integration dialog.
 *
 * Agent outputs: [GitHub](integration:github)
 */
export function IntegrationButton({ integrationName, label }: IntegrationButtonProps) {
  const displayName = label || formatIntegrationName(integrationName);

  function handleClick() {
    window.dispatchEvent(
      new CustomEvent("agent:open-integration", {
        detail: { integrationName },
      }),
    );
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md",
        "border border-slate-200 bg-white shadow-sm",
        "text-xs font-medium text-slate-700",
        "hover:bg-slate-50 hover:border-slate-300 hover:shadow",
        "transition-all cursor-pointer",
        "align-middle",
      )}
      title={`Open ${displayName} integration`}
    >
      <IntegrationIcon integrationName={integrationName} className="h-4 w-4" size={16} />
      <span>{displayName}</span>
    </button>
  );
}

/** Capitalize and clean up integration names for display */
function formatIntegrationName(name: string): string {
  // Handle dotted names like "aws.lambda" -> "AWS Lambda"
  return name
    .split(".")
    .map((part) => {
      if (part.length <= 3) return part.toUpperCase(); // aws -> AWS, sqs -> SQS
      return part.charAt(0).toUpperCase() + part.slice(1); // github -> Github
    })
    .join(" ");
}
