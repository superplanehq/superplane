import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { useReportPageReady } from "@/hooks/useReportPageReady";

import AdminPagination from "./AdminPagination";
import {
  LINEAR_WEBHOOK_PAGE_SIZE,
  LINEAR_WEBHOOKS_EMPTY,
  LINEAR_WEBHOOKS_HELP,
  LINEAR_WEBHOOKS_PAGE_EMPTY,
  linearWebhookIssueLabel,
  linearWebhookOutcomeLabel,
  type LinearWebhookReceipt,
} from "./linearWebhookReceipts";
import { useLinearWebhooks } from "./useLinearWebhooks";

export function LinearWebhooks() {
  const pageState = useLinearWebhooks();
  useReportPageReady(!pageState.loading || pageState.data !== null || pageState.loadError !== "");

  if (pageState.loading && pageState.data === null && pageState.loadError === "") {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading Linear webhooks...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{LINEAR_WEBHOOKS_HELP}</Text>
      <LinearWebhooksBody pageState={pageState} />
    </div>
  );
}

function LinearWebhooksBody({ pageState }: { pageState: ReturnType<typeof useLinearWebhooks> }) {
  if (pageState.loadError) {
    return <LinearWebhooksNotice message={pageState.loadError} />;
  }

  const items = pageState.data?.items ?? [];
  if (items.length === 0) {
    if (pageState.offset === 0) {
      return <LinearWebhooksNotice message={LINEAR_WEBHOOKS_EMPTY} />;
    }
    return (
      <div className="space-y-4">
        <LinearWebhooksNotice message={LINEAR_WEBHOOKS_PAGE_EMPTY} />
        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => pageState.setOffset(Math.max(0, pageState.offset - LINEAR_WEBHOOK_PAGE_SIZE))}
            className="px-3 py-1 rounded border border-slate-200 bg-white text-xs hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
          >
            Previous
          </button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <LinearWebhooksTable items={items} />
      <AdminPagination
        offset={pageState.offset}
        total={pageState.data?.total ?? 0}
        pageSize={LINEAR_WEBHOOK_PAGE_SIZE}
        onPageChange={pageState.setOffset}
      />
    </div>
  );
}

function LinearWebhooksNotice({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed border-slate-300 px-4 py-8 text-center dark:border-gray-700">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{message}</Text>
    </div>
  );
}

function LinearWebhookTaskIDs({ ids }: { ids?: string[] }) {
  if (!ids || ids.length === 0) {
    return <>—</>;
  }
  return (
    <div className="space-y-1">
      {ids.map((id) => (
        <div key={id} className="break-all">
          {id}
        </div>
      ))}
    </div>
  );
}

function LinearWebhooksTable({ items }: { items: LinearWebhookReceipt[] }) {
  return (
    <div className="overflow-x-auto rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Time</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Event</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Action</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Issue</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Team</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Workspace</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Result</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Task</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">HTTP status</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Integration</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Subscriptions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id} className="border-b border-slate-100 last:border-0 dark:border-gray-700/70">
              <td className="px-4 py-2.5">
                <Timestamp
                  date={item.received_at}
                  display="relative"
                  className="whitespace-nowrap text-gray-600 dark:text-gray-400"
                />
              </td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.event_type || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.action || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{linearWebhookIssueLabel(item) || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.team_key || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.workspace_key || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">
                {linearWebhookOutcomeLabel(item.outcome)}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">
                <LinearWebhookTaskIDs ids={item.task_ids} />
              </td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.http_status}</td>
              <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">
                {item.integration_id || "—"}
              </td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.subscription_count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
