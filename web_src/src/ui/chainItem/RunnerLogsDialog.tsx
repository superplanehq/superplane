import { useCallback, useEffect, useRef, useState } from "react";
import { canvasesGetRunnerExecutionLogs } from "@/api-client";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

interface RunnerLogsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  canvasId: string;
  executionId: string;
}

export function RunnerLogsDialog({ open, onOpenChange, canvasId, executionId }: RunnerLogsDialogProps) {
  const [lines, setLines] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const scrollRef = useRef<HTMLPreElement>(null);
  const nextTokenRef = useRef<string | undefined>(undefined);

  const appendEvents = useCallback((messages: string[]) => {
    if (messages.length === 0) {
      return;
    }
    setLines((prev) => [...prev, ...messages]);
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }

    setLines([]);
    setError(null);
    nextTokenRef.current = undefined;

    let cancelled = false;

    const poll = async () => {
      try {
        const res = await canvasesGetRunnerExecutionLogs(
          withOrganizationHeader({
            path: { canvasId, executionId },
            query:
              nextTokenRef.current !== undefined && nextTokenRef.current !== ""
                ? { nextForwardToken: nextTokenRef.current }
                : {},
          }),
        );

        if (cancelled) {
          return;
        }

        const data = res.data;
        const events = data?.events ?? [];
        const messages = events.map((e) => e.message ?? "").filter((m) => m.length > 0);
        appendEvents(messages);

        if (data?.nextForwardToken) {
          nextTokenRef.current = data.nextForwardToken;
        }

        setError(null);
      } catch (e) {
        if (!cancelled) {
          setError(getApiErrorMessage(e));
        }
      }
    };

    void poll();
    const interval = window.setInterval(() => {
      void poll();
    }, 2000);

    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, [open, canvasId, executionId, appendEvents]);

  useEffect(() => {
    if (!open || !scrollRef.current) {
      return;
    }
    const el = scrollRef.current;
    el.scrollTop = el.scrollHeight;
  }, [lines, open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="large" className="flex h-full max-h-[90vh] w-[90vw] flex-col gap-0 overflow-hidden p-0">
        <div className="flex h-full min-h-0 flex-col">
          <div className="flex shrink-0 items-center border-b border-gray-200 bg-white px-4 py-3 pr-12">
            <DialogTitle className="text-left text-base font-semibold text-gray-900">Runner logs</DialogTitle>
          </div>

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-slate-50 p-4">
            <pre
              ref={scrollRef}
              className="min-h-0 w-full flex-1 overflow-auto rounded-md border border-slate-200 bg-white p-4 text-left text-xs leading-relaxed font-mono text-gray-800 whitespace-pre-wrap break-words"
            >
              {error ? (
                <span className="text-destructive">{error}</span>
              ) : lines.length === 0 ? (
                <span className="text-gray-500">Waiting for log output…</span>
              ) : (
                lines.join("")
              )}
            </pre>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
