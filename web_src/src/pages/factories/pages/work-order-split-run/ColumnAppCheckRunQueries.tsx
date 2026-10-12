import type { CanvasesCanvasRun } from "@/api-client";
import { useDescribeRun } from "@/hooks/useCanvasData";
import { useEffect } from "react";

import type { ColumnAppCheckRunTarget } from "./splitRunMocks";

export function ColumnAppCheckRunQueries({
  targets,
  onRun,
}: {
  targets: ColumnAppCheckRunTarget[];
  onRun: (runId: string, run: CanvasesCanvasRun | undefined, loading: boolean) => void;
}) {
  return (
    <>
      {targets.map((target) => (
        <ColumnAppCheckRunQuery key={`${target.appId}:${target.runId}`} target={target} onRun={onRun} />
      ))}
    </>
  );
}

function ColumnAppCheckRunQuery({
  target,
  onRun,
}: {
  target: ColumnAppCheckRunTarget;
  onRun: (runId: string, run: CanvasesCanvasRun | undefined, loading: boolean) => void;
}) {
  const query = useDescribeRun(target.appId, target.runId);
  useEffect(() => {
    onRun(target.runId, query.data?.run, query.isLoading);
  }, [onRun, query.data?.run, query.isLoading, target.runId]);
  return null;
}
