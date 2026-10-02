import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { useReportPageReady } from "@/hooks/useReportPageReady";

import AdminPagination from "./AdminPagination";
import {
  DATADOG_WEBHOOK_PAGE_SIZE,
  DATADOG_WEBHOOKS_EMPTY,
  DATADOG_WEBHOOKS_HELP,
  DATADOG_WEBHOOKS_PAGE_EMPTY,
  datadogWebhookOutcomeLabel,
  type DatadogWebhookReceipt,
} from "./datadogWebhookReceipts";
import { useDatadogWebhooks } from "./useDatadogWebhooks";

export function DatadogWebhooks() {
  const pageState = useDatadogWebhooks();
  useReportPageReady(!pageState.loading || pageState.data !== null || pageState.loadError !== "");

  if (pageState.loading && pageState.data === null && pageState.loadError === "") {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading Datadog webhooks...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{DATADOG_WEBHOOKS_HELP}</Text>
      <DatadogWebhooksBody pageState={pageState} />
    </div>
  );
}

function DatadogWebhooksBody({ pageState }: { pageState: ReturnType<typeof useDatadogWebhooks> }) {
  if (pageState.loadError) {
    return <DatadogWebhooksNotice message={pageState.loadError} />;
  }

  const items = pageState.data?.items ?? [];
  if (items.length === 0) {
    if (pageState.offset === 0) {
      return <DatadogWebhooksNotice message={DATADOG_WEBHOOKS_EMPTY} />;
    }
    return (
      <div className="space-y-4">
        <DatadogWebhooksNotice message={DATADOG_WEBHOOKS_PAGE_EMPTY} />
        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => pageState.setOffset(Math.max(0, pageState.offset - DATADOG_WEBHOOK_PAGE_SIZE))}
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
      <DatadogWebhooksTable items={items} />
      <AdminPagination
        offset={pageState.offset}
        total={pageState.data?.total ?? 0}
        pageSize={DATADOG_WEBHOOK_PAGE_SIZE}
        onPageChange={pageState.setOffset}
      />
    </div>
  );
}

function DatadogWebhooksNotice({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed border-slate-300 px-4 py-8 text-center dark:border-gray-700">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{message}</Text>
    </div>
  );
}

function DatadogWebhookTaskIDs({ ids }: { ids?: string[] }) {
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

function DatadogWebhooksTable({ items }: { items: DatadogWebhookReceipt[] }) {
  return (
    <div className="overflow-x-auto rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Time</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Event</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Transition</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Service</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Alert</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Issue</th>
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
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.alert_transition || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.service || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.alert_id || "—"}</td>
              <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">{item.issue_id || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">
                {datadogWebhookOutcomeLabel(item.outcome)}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">
                <DatadogWebhookTaskIDs ids={item.task_ids} />
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
