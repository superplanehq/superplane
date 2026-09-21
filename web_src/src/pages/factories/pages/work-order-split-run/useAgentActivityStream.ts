import { useEffect, useState } from "react";

import { LiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/liveLogStream";

import {
  emptyAgentActivityState,
  reduceAgentActivityRecords,
  type AgentActivity,
  type AgentActivityRecord,
  type AgentActivityState,
} from "./agentActivity";

const RECONNECT_DELAY_MS = 2000;

type AgentActivityStreamResult = {
  activities: AgentActivity[];
  error?: string;
  isConnected: boolean;
  hasConnectedOnce: boolean;
};

type StreamConnection = Omit<AgentActivityStreamResult, "activities"> & { executionId?: string };

function disconnectedConnection(current: StreamConnection, executionId: string, error: string): StreamConnection {
  return {
    executionId,
    error,
    isConnected: false,
    hasConnectedOnce: current.executionId === executionId && current.hasConnectedOnce,
  };
}

export function useAgentActivityStream({
  organizationId,
  canvasId,
  executionId,
  active,
}: {
  organizationId?: string;
  canvasId?: string;
  executionId?: string;
  active: boolean;
}): AgentActivityStreamResult {
  const [scopedState, setScopedState] = useState<{ executionId?: string; activity: AgentActivityState }>({
    executionId,
    activity: emptyAgentActivityState,
  });
  const [connection, setConnection] = useState<StreamConnection>({
    executionId,
    isConnected: false,
    hasConnectedOnce: false,
  });
  const state = scopedState.executionId === executionId ? scopedState.activity : emptyAgentActivityState;
  const currentConnection =
    connection.executionId === executionId ? connection : { isConnected: false, hasConnectedOnce: false };

  useEffect(() => {
    if (!organizationId || !canvasId || !executionId || !active) {
      return;
    }

    const abortController = new AbortController();
    const pendingRecords: AgentActivityRecord[] = [];
    let stream: LiveLogStream | undefined;
    let animationFrame: number | undefined;

    const flushRecords = () => {
      animationFrame = undefined;
      if (pendingRecords.length === 0) {
        return;
      }
      const records = pendingRecords.splice(0, pendingRecords.length);
      setScopedState((current) => ({
        executionId,
        activity: reduceAgentActivityRecords(
          current.executionId === executionId ? current.activity : emptyAgentActivityState,
          records,
        ),
      }));
    };
    const queueRecord = (record: AgentActivityRecord) => {
      pendingRecords.push(record);
      if (animationFrame === undefined) {
        animationFrame = window.requestAnimationFrame(flushRecords);
      }
    };
    const waitToReconnect = () =>
      new Promise<void>((resolve) => {
        const timeout = window.setTimeout(resolve, RECONNECT_DELAY_MS);
        abortController.signal.addEventListener(
          "abort",
          () => {
            window.clearTimeout(timeout);
            resolve();
          },
          { once: true },
        );
      });

    const run = async () => {
      while (!abortController.signal.aborted) {
        const currentStream = new LiveLogStream(organizationId, canvasId, executionId);
        stream = currentStream;
        try {
          await currentStream.pump({
            onOpen: () => {
              setConnection({ executionId, isConnected: true, hasConnectedOnce: true });
            },
            onRecord: queueRecord,
            onLogLine: () => undefined,
            onStreamError: (message) =>
              setConnection((current) => disconnectedConnection(current, executionId, message)),
          });
        } catch (streamError) {
          if (!abortController.signal.aborted) {
            const message = streamError instanceof Error ? streamError.message : String(streamError);
            setConnection((current) => disconnectedConnection(current, executionId, message));
          }
        } finally {
          currentStream.stop();
          if (stream === currentStream) {
            stream = undefined;
          }
          if (!abortController.signal.aborted) {
            setConnection((current) =>
              current.executionId === executionId ? { ...current, isConnected: false } : current,
            );
          }
        }
        if (!abortController.signal.aborted) {
          await waitToReconnect();
        }
      }
    };

    void run();
    return () => {
      abortController.abort();
      stream?.stop();
      if (animationFrame !== undefined) {
        window.cancelAnimationFrame(animationFrame);
      }
      pendingRecords.length = 0;
    };
  }, [active, canvasId, executionId, organizationId]);

  return {
    activities: state.activities,
    error: currentConnection.error,
    isConnected: currentConnection.isConnected,
    hasConnectedOnce: currentConnection.hasConnectedOnce,
  };
}
