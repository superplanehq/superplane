import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useEffect, useState } from "react";
import { LiveLogStream } from "./liveLogStream";
import { useScrollToBottom } from "./useScrollToBottom";

export function useLiveLogStream(executionId: string) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [lines, setLines] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);

  const text = lines.join("");
  const { scrollRef } = useScrollToBottom(text);

  useEffect(() => {
    if (!organizationId || !canvasId || !executionId) {
      return;
    }

    setLines([]);
    setError(null);
    setIsStreaming(true);

    const stream = new LiveLogStream(organizationId, canvasId, executionId);

    (async () => {
      try {
        await stream.pump({
          onLogLine: (t) => setLines((ls) => [...ls, t]),
          onStreamError: (m) => setError(m),
        });
      } catch (e) {
        if ((e as Error).name === "AbortError") {
          return;
        }
        setError((e as Error).message);
      } finally {
        setIsStreaming(false);
      }
    })();

    return () => stream.stop();
  }, [organizationId, canvasId, executionId]);

  return { lines, text, error, isStreaming, scrollRef };
}
