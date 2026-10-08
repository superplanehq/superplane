import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useEffect, useRef, useState, type ChangeEvent, type FormEvent } from "react";

import { type FleetRunner, type FleetTask, fetchRunner, nextCursor, previousCursor } from "./fleetAdmin";
import { PageControls, RelativeTimestamp, StateBadge, headerCellClass, rowClass, tableClass } from "./fleetTable";

export const RunnerRows = ({ rows }: { rows: FleetRunner[] }) => (
  <div className={tableClass}>
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-slate-100 dark:border-gray-700/70">
          <th className={headerCellClass}>ID</th>
          <th className={headerCellClass}>State</th>
          <th className={headerCellClass}>Runner version</th>
          <th className={headerCellClass}>OS</th>
          <th className={headerCellClass}>Architecture</th>
          <th className={headerCellClass}>Hostname</th>
          <th className={headerCellClass}>IP address</th>
          <th className={headerCellClass}>Tags</th>
          <th className={headerCellClass}>Ephemeral</th>
          <th className={headerCellClass}>Created</th>
          <th className={headerCellClass}>Last seen</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((runner) => (
          <tr key={runner.id} className={rowClass}>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">{runner.id}</td>
            <td className="px-4 py-2.5">
              <StateBadge state={runner.state} />
            </td>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">
              {runner.runnerVersion?.trim() || "—"}
            </td>
            <td className="px-4 py-2.5 text-xs text-gray-700 dark:text-gray-300">{runner.os || "—"}</td>
            <td className="px-4 py-2.5 text-xs text-gray-700 dark:text-gray-300">{runner.arch || "—"}</td>
            <td className="px-4 py-2.5 text-xs text-gray-700 dark:text-gray-300">{runner.hostname || "—"}</td>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">{runner.ip || "—"}</td>
            <td className="px-4 py-2.5 text-xs text-gray-700 dark:text-gray-300">
              {Object.entries(runner.tags ?? {}).length === 0 ? (
                "—"
              ) : (
                <dl className="space-y-1">
                  {Object.entries(runner.tags ?? {})
                    .sort(([a], [b]) => a.localeCompare(b))
                    .map(([key, value]) => (
                      <div key={key} className="flex gap-1">
                        <dt className="font-medium">{key}:</dt>
                        <dd className="font-mono break-all">{value}</dd>
                      </div>
                    ))}
                </dl>
              )}
            </td>
            <td className="px-4 py-2.5 text-gray-700 dark:text-gray-300">{runner.ephemeral ? "Yes" : "No"}</td>
            <td className="px-4 py-2.5">
              <RelativeTimestamp value={runner.createdAt} />
            </td>
            <td className="px-4 py-2.5">
              <RelativeTimestamp value={runner.lastSeenAt} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

export const TaskRows = ({ rows }: { rows: FleetTask[] }) => (
  <div className={tableClass}>
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-slate-100 dark:border-gray-700/70">
          <th className={headerCellClass}>ID</th>
          <th className={headerCellClass}>Organization</th>
          <th className={headerCellClass}>State</th>
          <th className={headerCellClass}>Runner</th>
          <th className={headerCellClass}>Queued</th>
          <th className={headerCellClass}>Started</th>
          <th className={headerCellClass}>Finished</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((task) => (
          <tr key={task.id} className={rowClass}>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">{task.id}</td>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">
              {task.organizationId || "—"}
            </td>
            <td className="px-4 py-2.5">
              <StateBadge state={task.state} />
            </td>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">
              {task.runnerId?.trim() || "—"}
            </td>
            <td className="px-4 py-2.5">
              <RelativeTimestamp value={task.queuedAt} />
            </td>
            <td className="px-4 py-2.5">
              <RelativeTimestamp value={task.startedAt} />
            </td>
            <td className="px-4 py-2.5">
              <RelativeTimestamp value={task.finishedAt} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const EmptyRecords = ({ message }: { message: string }) => (
  <div className="rounded-xl border border-slate-200 bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
    <Text className="text-sm text-gray-600 dark:text-gray-400">{message}</Text>
  </div>
);

export const RunnerList = ({
  fleetId,
  runners,
  cursors,
  setCursors,
}: {
  fleetId: string;
  runners: { rows: FleetRunner[]; totalCount: number; hasNextPage: boolean } | null;
  cursors: string[];
  setCursors: (update: (current: string[]) => string[]) => void;
}) => {
  const [runnerId, setRunnerId] = useState("");
  const [foundRunner, setFoundRunner] = useState<FleetRunner | null>(null);
  const [lookupError, setLookupError] = useState("");
  const lookupAbort = useRef<AbortController | null>(null);

  useEffect(() => () => lookupAbort.current?.abort(), []);

  const findRunner = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    lookupAbort.current?.abort();
    setFoundRunner(null);
    setLookupError("");
    const id = runnerId.trim();
    if (!id) {
      return;
    }
    const abort = new AbortController();
    lookupAbort.current = abort;
    try {
      setFoundRunner(await fetchRunner(fleetId, id, abort.signal));
    } catch (error) {
      if (!abort.signal.aborted) {
        setLookupError(error instanceof Error ? error.message : "Failed to load runner.");
      }
    }
  };

  if (!runners) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading runners...</Text>;
  }

  return (
    <>
      <form onSubmit={findRunner} className="mb-3 flex items-end gap-2">
        <div className="w-full max-w-md">
          <label htmlFor="runner-id-lookup" className="mb-1 block text-sm text-gray-700 dark:text-gray-300">
            Runner ID
          </label>
          <Input
            id="runner-id-lookup"
            value={runnerId}
            onChange={(event: ChangeEvent<HTMLInputElement>) => setRunnerId(event.target.value)}
          />
        </div>
        <Button type="submit" disabled={!runnerId.trim()}>
          Find runner
        </Button>
      </form>
      {lookupError && <Text className="mb-3 text-sm text-red-600 dark:text-red-400">{lookupError}</Text>}
      {foundRunner && (
        <div className="mb-3">
          <RunnerRows rows={[foundRunner]} />
        </div>
      )}
      {runners.rows.length === 0 ? (
        <EmptyRecords message="No idle or busy runners." />
      ) : (
        <RunnerRows rows={runners.rows} />
      )}
      <PageControls
        pageIndex={cursors.length}
        rowCount={runners.rows.length}
        totalCount={runners.totalCount}
        hasNextPage={runners.hasNextPage}
        onPrevious={() => setCursors(previousCursor)}
        onNext={() => setCursors((current) => nextCursor(current, runners.rows.at(-1)?.id, runners.hasNextPage))}
        previousTestId="fleet-runners-previous"
        nextTestId="fleet-runners-next"
      />
    </>
  );
};

export const TaskList = ({
  tasks,
  cursors,
  setCursors,
}: {
  tasks: { rows: FleetTask[]; totalCount: number; hasNextPage: boolean } | null;
  cursors: string[];
  setCursors: (update: (current: string[]) => string[]) => void;
}) => {
  if (!tasks) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading tasks...</Text>;
  }

  return (
    <>
      {tasks.rows.length === 0 ? (
        <EmptyRecords message="No queued, reserved, or running tasks." />
      ) : (
        <TaskRows rows={tasks.rows} />
      )}
      <PageControls
        pageIndex={cursors.length}
        rowCount={tasks.rows.length}
        totalCount={tasks.totalCount}
        hasNextPage={tasks.hasNextPage}
        onPrevious={() => setCursors(previousCursor)}
        onNext={() => setCursors((current) => nextCursor(current, tasks.rows.at(-1)?.id, tasks.hasNextPage))}
        previousTestId="fleet-tasks-previous"
        nextTestId="fleet-tasks-next"
      />
    </>
  );
};
