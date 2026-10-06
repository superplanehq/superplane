import { Text } from "@/components/Text/text";
import { Terminal } from "lucide-react";

import {
  REFRESH_INTERVAL_MS,
  formatBrokerStatus,
  formatExecutionMode,
  statusBadgeClass,
  type BrokerTask,
} from "./fleetAdmin";
import { RelativeTimestamp, headerCellClass, rowClass, tableClass } from "./fleetTable";

const BrokerTaskTable = ({ tasks }: { tasks: BrokerTask[] }) => (
  <div className={tableClass}>
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-slate-100 dark:border-gray-700/70">
          <th className={headerCellClass}>Task ID</th>
          <th className={headerCellClass}>Status</th>
          <th className={headerCellClass}>Fleet</th>
          <th className={headerCellClass}>Runner</th>
          <th className={headerCellClass}>Execution</th>
          <th className={headerCellClass}>Created</th>
          <th className={headerCellClass}>Claimed</th>
          <th className={headerCellClass}>Lease until</th>
        </tr>
      </thead>
      <tbody>
        {tasks.map((task) => (
          <BrokerTaskRow key={task.id} task={task} />
        ))}
      </tbody>
    </table>
  </div>
);

const BrokerTaskRow = ({ task }: { task: BrokerTask }) => (
  <tr className={`${rowClass} transition-colors`}>
    <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100" title={task.id}>
      {task.id}
    </td>
    <td className="px-4 py-2.5">
      <span
        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeClass(task.status, task.cancel_requested ?? false)}`}
      >
        {formatBrokerStatus(task.status, task.cancel_requested ?? false)}
      </span>
    </td>
    <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">{task.fleet_id || "—"}</td>
    <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">{task.runner_id?.trim() || "—"}</td>
    <td className="px-4 py-2.5 text-gray-700 dark:text-gray-300">{formatExecutionMode(task)}</td>
    <td className="px-4 py-2.5">
      <RelativeTimestamp value={task.created_at} />
    </td>
    <td className="px-4 py-2.5">
      <RelativeTimestamp value={task.claimed_at} />
    </td>
    <td className="px-4 py-2.5">
      <RelativeTimestamp value={task.lease_until} />
    </td>
  </tr>
);

const BrokerState = ({ configured, tasks }: { configured: boolean | null; tasks: BrokerTask[] }) => {
  if (configured === false) {
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
        <Terminal size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
        <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">
          Runner task broker is not configured. Set{" "}
          <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-xs dark:bg-gray-800 dark:text-gray-200">
            TASK_BROKER_BASE_URL
          </code>{" "}
          and{" "}
          <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-xs dark:bg-gray-800 dark:text-gray-200">
            TASK_BROKER_AUTH_TOKEN
          </code>{" "}
          on the app server.
        </Text>
      </div>
    );
  }
  if (configured && tasks.length === 0) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
        <Terminal size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
        <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">No active runner tasks right now.</Text>
      </div>
    );
  }
  if (!configured) {
    return null;
  }
  return <BrokerTaskTable tasks={tasks} />;
};

export const BrokerTaskSection = ({ configured, tasks }: { configured: boolean | null; tasks: BrokerTask[] }) => (
  <section className="space-y-4" data-testid="task-broker-section">
    <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Task broker</h2>
        <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Active tasks on the task broker (queued or claimed). Refreshes every {REFRESH_INTERVAL_MS / 1000} seconds.
        </Text>
      </div>
      <Text className="text-xs text-gray-500 dark:text-gray-400">
        {tasks.length} active task{tasks.length === 1 ? "" : "s"}
      </Text>
    </div>
    <BrokerState configured={configured} tasks={tasks} />
  </section>
);
