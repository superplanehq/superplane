import { useLiveLogStream } from "./useLiveLogStream";

export function LiveLogStreamView({ executionId }: { executionId: string }) {
  const { text, error, isStreaming, scrollRef } = useLiveLogStream(executionId);

  return (
    <div className="flex min-h-[50vh] flex-col overflow-hidden bg-slate-50">
      <pre
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-auto p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap text-left text-gray-800"
      >
        {error ? <span className="text-destructive">{error}</span> : null}
        {!error && !text && !isStreaming ? <span className="text-muted-foreground">No log lines yet.</span> : null}
        {!error && text}
        {!error && isStreaming && !text ? <span className="text-muted-foreground">Connecting…</span> : null}
      </pre>
    </div>
  );
}
