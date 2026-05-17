import { useLiveLogStream } from "./useLiveLogStream";

export function LiveLogStreamView({ executionId }: { executionId: string }) {
  const { lines, error, isStreaming, scrollRef } = useLiveLogStream(executionId);

  return (
    <div className="flex min-h-[50vh] flex-col overflow-hidden bg-slate-50">
      <pre
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-auto p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap text-left text-gray-800"
      >
        {error ? (
          <span className="text-destructive">{error}</span>
        ) : null}

        {!error && lines.length === 0 && !isStreaming ? (
          <span className="text-muted-foreground">No log lines yet.</span>
        ) : null}

        {!error && lines.length === 0 && isStreaming ? (
          <span className="text-muted-foreground">Connecting…</span>
        ) : null}

        {!error &&
          lines.map((line, i) => {
            const isCommand = line.trimStart().startsWith("+ ");
            return (
              <span key={i} className="block">
                {isCommand ? (
                  <span className="mr-2 inline-flex items-center rounded px-1 py-0.5 text-[10px] font-bold bg-green-100 text-green-700">
                    ✓ CMD
                  </span>
                ) : null}
                {line}
              </span>
            );
          })}
      </pre>
    </div>
  );
}