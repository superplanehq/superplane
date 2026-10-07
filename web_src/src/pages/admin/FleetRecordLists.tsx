import { Text } from "@/components/Text/text";

import { type FleetRunner, type FleetTask, nextCursor, previousCursor } from "./fleetAdmin";
import { PageControls, RelativeTimestamp, StateBadge, headerCellClass, rowClass, tableClass } from "./fleetTable";

export const RunnerRows = ({ rows }: { rows: FleetRunner[] }) => (
  <div className={tableClass}>
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-slate-100 dark:border-gray-700/70">
          <th className={headerCellClass}>ID</th>
          <th className={headerCellClass}>State</th>
          <th className={headerCellClass}>Runner version</th>
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
  runners,
  cursors,
  setCursors,
}: {
  runners: { rows: FleetRunner[]; totalCount: number; hasNextPage: boolean } | null;
  cursors: string[];
  setCursors: (update: (current: string[]) => string[]) => void;
}) => {
  if (!runners) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading runners...</Text>;
  }

  return (
    <>
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
