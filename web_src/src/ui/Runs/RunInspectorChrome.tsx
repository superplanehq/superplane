import { ChevronDown, ChevronUp, ChevronsRight, Link as LinkIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

export function RunInspectorChrome({
  runId,
  newerRunId,
  olderRunId,
  canNavigateOlder,
  onNavigateRun,
  onNavigateOlder,
  onClose,
}: {
  runId?: string | null;
  newerRunId?: string | null;
  olderRunId?: string | null;
  canNavigateOlder?: boolean;
  onNavigateRun?: (runId: string) => void;
  onNavigateOlder?: () => void;
  onClose: () => void;
}) {
  const canUseOlderNavigation = canNavigateOlder ?? Boolean(olderRunId);
  const olderNavigationDisabled = !canUseOlderNavigation || (olderRunId ? !onNavigateRun : !onNavigateOlder);

  const handleOlderNavigation = () => {
    if (olderRunId) {
      onNavigateRun?.(olderRunId);
      return;
    }

    onNavigateOlder?.();
  };

  return (
    <div className="flex shrink-0 items-center justify-between gap-2 border-b border-slate-950/10 px-2 py-1.5 dark:border-gray-800">
      <div className="flex items-center gap-1">
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              aria-label="Close"
              onClick={onClose}
              className="flex h-7 w-7 items-center justify-center rounded text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
              data-testid="run-panel-close"
            >
              <ChevronsRight className="h-4 w-4" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Close</TooltipContent>
        </Tooltip>
        <RunNavigationButton
          label="Newer run"
          disabled={!newerRunId || !onNavigateRun}
          onClick={() => newerRunId && onNavigateRun?.(newerRunId)}
        >
          <ChevronUp className="h-4 w-4" />
        </RunNavigationButton>
        <RunNavigationButton label="Older run" disabled={olderNavigationDisabled} onClick={handleOlderNavigation}>
          <ChevronDown className="h-4 w-4" />
        </RunNavigationButton>
      </div>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <button
              type="button"
              aria-label="Copy run link"
              disabled={!runId}
              onClick={() => runId && void copyRunLink(runId)}
              className={chromeIconButtonClassName}
            >
              <LinkIcon className="h-4 w-4" />
            </button>
          </span>
        </TooltipTrigger>
        <TooltipContent side="bottom">Copy link</TooltipContent>
      </Tooltip>
    </div>
  );
}

function RunNavigationButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string;
  disabled: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <button
            type="button"
            aria-label={label}
            disabled={disabled}
            onClick={onClick}
            className={chromeIconButtonClassName}
          >
            {children}
          </button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}

const chromeIconButtonClassName = cn(
  "flex h-7 w-7 items-center justify-center rounded text-gray-500 transition-colors",
  "hover:bg-gray-100 hover:text-gray-800 disabled:cursor-not-allowed disabled:opacity-40",
  "dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
);

async function copyRunLink(runId: string) {
  const url = new URL(window.location.href);
  url.search = "";
  url.searchParams.set("run", runId);

  try {
    await navigator.clipboard.writeText(url.toString());
    toast.success("Run link copied");
  } catch {
    toast.error("Failed to copy run link");
  }
}
