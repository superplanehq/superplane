import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CircleAlert, Loader2, Radio, RefreshCw, Terminal } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

type NonErrorLiveLogState = "loading" | "waiting" | "empty";

type LiveLogStateNoticeProps =
  | {
      state: NonErrorLiveLogState;
      compact?: boolean;
    }
  | {
      state: "error";
      error: string;
      willRetry: boolean;
      onRetry: () => void;
      compact?: boolean;
    };

const liveLogStateContent: Record<
  NonErrorLiveLogState,
  { title: string; description: string; Icon: LucideIcon; iconClassName?: string }
> = {
  loading: {
    title: "Loading logs",
    description: "SuperPlane is loading saved log output.",
    Icon: Loader2,
    iconClassName: "animate-spin",
  },
  waiting: {
    title: "Waiting for logs",
    description: "Logs will appear when the runner sends output.",
    Icon: Radio,
    iconClassName: "animate-pulse",
  },
  empty: {
    title: "No logs available",
    description: "This execution did not produce log output.",
    Icon: Terminal,
  },
};

export function LiveLogStateNotice(props: LiveLogStateNoticeProps) {
  const compact = props.compact ?? false;

  if (props.state === "error") {
    return (
      <div
        role="alert"
        className={cn(
          "flex flex-col items-center justify-center gap-2 px-6 text-center",
          compact ? "py-4" : "min-h-48 py-8",
        )}
      >
        <StateIcon tone="error">
          <CircleAlert className="size-5" aria-hidden />
        </StateIcon>
        <div className="space-y-1">
          <p className="text-sm font-medium text-red-700 dark:text-red-300">
            {props.willRetry ? "Logs are temporarily unavailable" : "Could not load logs"}
          </p>
          <p className="text-xs text-slate-500 dark:text-gray-400">
            {props.willRetry ? "SuperPlane will try again automatically." : "Check your connection and try again."}
          </p>
        </div>
        <details className="max-w-xl text-left text-xs text-slate-500 dark:text-gray-400">
          <summary className="cursor-pointer select-none text-center hover:text-slate-700 dark:hover:text-gray-200">
            Error details
          </summary>
          <p className="mt-2 max-h-28 overflow-auto rounded-md bg-red-50 px-3 py-2 font-mono break-words text-red-700 dark:bg-red-950/30 dark:text-red-300">
            {props.error}
          </p>
        </details>
        <Button type="button" variant="outline" size="sm" onClick={props.onRetry}>
          <RefreshCw className="size-3.5" aria-hidden />
          Try again
        </Button>
      </div>
    );
  }

  const content = liveLogStateContent[props.state];
  return (
    <div
      role={props.state === "empty" ? undefined : "status"}
      className={cn(
        "flex flex-col items-center justify-center gap-2 px-6 text-center",
        compact ? "py-4" : "min-h-48 py-8",
      )}
    >
      <StateIcon tone={props.state}>
        <content.Icon className={cn("size-5", content.iconClassName)} aria-hidden />
      </StateIcon>
      <div className="space-y-1">
        <p className="text-sm font-medium text-slate-700 dark:text-gray-200">{content.title}</p>
        <p className="text-xs text-slate-500 dark:text-gray-400">{content.description}</p>
      </div>
    </div>
  );
}

function StateIcon({ tone, children }: { tone: "loading" | "waiting" | "empty" | "error"; children: ReactNode }) {
  return (
    <div
      className={cn("flex size-10 items-center justify-center rounded-full", {
        "bg-blue-100 text-blue-600 dark:bg-blue-950/50 dark:text-blue-300": tone === "loading",
        "bg-amber-100 text-amber-600 dark:bg-amber-950/50 dark:text-amber-300": tone === "waiting",
        "bg-slate-200 text-slate-500 dark:bg-gray-800 dark:text-gray-400": tone === "empty",
        "bg-red-100 text-red-600 dark:bg-red-950/50 dark:text-red-300": tone === "error",
      })}
    >
      {children}
    </div>
  );
}
