import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { showErrorToast } from "@/lib/toast";
import { Terminal } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";

type RunnerTask = {
  id: string;
  status: string;
  fleet_id: string;
  created_at: string;
  claimed_at?: string;
  lease_until?: string;
  runner_id?: string;
  execution_mode?: string;
  docker_image?: string;
  cancel_requested?: boolean;
  execution_timeout_seconds?: number;
};

type RunnerTasksResponse = {
  configured: boolean;
  tasks: RunnerTask[];
};

const REFRESH_INTERVAL_MS = 5000;

const statusBadgeClass = (status: string, cancelRequested: boolean) => {
  if (cancelRequested) {
    return "bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300";
  }

  switch (status) {
    case "queued":
      return "bg-slate-100 text-slate-700 dark:bg-gray-800 dark:text-gray-300";
    case "claimed":
      return "bg-blue-100 text-blue-800 dark:bg-blue-950/40 dark:text-blue-300";
    default:
      return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300";
  }
};

const formatStatus = (status: string, cancelRequested: boolean) => {
  if (cancelRequested) {
    return "cancel requested";
  }

  return status;
};

const formatExecutionMode = (task: RunnerTask) => {
  const mode = task.execution_mode?.trim() || "host";
  if (mode === "docker" && task.docker_image) {
    return `docker (${task.docker_image})`;
  }

  return mode;
};

const RelativeTimestamp = ({ value }: { value?: string }) => (
  <Timestamp
    date={value}
    display="relative"
    className="text-gray-600 whitespace-nowrap dark:text-gray-400"
    fallback={<span className="text-gray-400 dark:text-gray-500">—</span>}
  />
);

const RunnerTasksTable = ({ tasks }: { tasks: RunnerTask[] }) => (
  <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-slate-100 dark:border-gray-700/70">
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Task ID</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Status</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Fleet</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Runner</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Execution</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Created</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Claimed</th>
          <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Lease until</th>
        </tr>
      </thead>
      <tbody>
        {tasks.map((task) => (
          <tr
            key={task.id}
            className="border-b border-slate-50 last:border-0 hover:bg-slate-50 transition-colors dark:border-gray-800/70 dark:hover:bg-gray-800/50"
          >
            <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100" title={task.id}>
              {task.id}
            </td>
            <td className="px-4 py-2.5">
              <span
                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeClass(task.status, task.cancel_requested ?? false)}`}
              >
                {formatStatus(task.status, task.cancel_requested ?? false)}
              </span>
            </td>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">{task.fleet_id || "—"}</td>
            <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">
              {task.runner_id?.trim() || "—"}
            </td>
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
        ))}
      </tbody>
    </table>
  </div>
);

const RunnerTasks: React.FC = () => {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [tasks, setTasks] = useState<RunnerTask[]>([]);
  const [loading, setLoading] = useState(true);

  const loadTasks = useCallback(async (showLoading: boolean) => {
    if (showLoading) {
      setLoading(true);
    }

    try {
      const response = await fetch("/admin/api/runner/tasks", { credentials: "include" });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text.trim() || "Failed to load runner tasks");
      }

      const data: RunnerTasksResponse = await response.json();
      setConfigured(data.configured);
      setTasks(data.tasks ?? []);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load runner tasks");
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void loadTasks(true);
    const interval = window.setInterval(() => {
      void loadTasks(false);
    }, REFRESH_INTERVAL_MS);

    return () => window.clearInterval(interval);
  }, [loadTasks]);

  useReportPageReady(!loading || configured !== null);

  if (loading && configured === null) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading runner tasks...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Runner Tasks</h1>
          <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Active tasks on the task broker (queued or claimed). Refreshes every {REFRESH_INTERVAL_MS / 1000} seconds.
          </Text>
        </div>
        <Text className="text-xs text-gray-500 dark:text-gray-400">
          {tasks.length} active task{tasks.length === 1 ? "" : "s"}
        </Text>
      </div>

      {!configured ? (
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
      ) : tasks.length === 0 ? (
        <div className="rounded-xl border border-slate-200 bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
          <Terminal size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
          <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">No active runner tasks right now.</Text>
        </div>
      ) : (
        <RunnerTasksTable tasks={tasks} />
      )}
    </div>
  );
};

export default RunnerTasks;
