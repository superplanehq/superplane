import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { useReportPageReady } from "@/hooks/useReportPageReady";

import AdminPagination from "./AdminPagination";
import {
  SENTRY_WEBHOOK_PAGE_SIZE,
  SENTRY_WEBHOOKS_EMPTY,
  SENTRY_WEBHOOKS_HELP,
  SENTRY_WEBHOOKS_TITLE,
  sentryWebhookIssueLabel,
  sentryWebhookOutcomeLabel,
  type SentryWebhookReceipt,
} from "./sentryWebhookReceipts";
import { useSentryWebhooks } from "./useSentryWebhooks";

export function SentryWebhooks() {
  const pageState = useSentryWebhooks();
  useReportPageReady(!pageState.loading || pageState.data !== null || pageState.loadError !== "");

  if (pageState.loading && pageState.data === null && pageState.loadError === "") {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading Sentry webhooks...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{SENTRY_WEBHOOKS_TITLE}</h1>
        <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">{SENTRY_WEBHOOKS_HELP}</Text>
      </div>
      <SentryWebhooksBody pageState={pageState} />
    </div>
  );
}

function SentryWebhooksBody({ pageState }: { pageState: ReturnType<typeof useSentryWebhooks> }) {
  if (pageState.loadError) {
    return <SentryWebhooksNotice message={pageState.loadError} />;
  }

  const items = pageState.data?.items ?? [];
  if (items.length === 0) {
    return <SentryWebhooksNotice message={SENTRY_WEBHOOKS_EMPTY} />;
  }

  return (
    <div>
      <SentryWebhooksTable items={items} />
      <AdminPagination
        offset={pageState.offset}
        total={pageState.data?.total ?? 0}
        pageSize={SENTRY_WEBHOOK_PAGE_SIZE}
        onPageChange={pageState.setOffset}
      />
    </div>
  );
}

function SentryWebhooksNotice({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed border-slate-300 px-4 py-8 text-center dark:border-gray-700">
      <Text className="text-sm text-gray-500 dark:text-gray-400">{message}</Text>
    </div>
  );
}

function SentryWebhooksTable({ items }: { items: SentryWebhookReceipt[] }) {
  return (
    <div className="overflow-hidden rounded-md bg-white shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Time</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Resource</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Action</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Project</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Issue</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Result</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">HTTP status</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Installation</th>
            <th className="px-4 py-2.5 text-left font-medium text-gray-500 dark:text-gray-400">Connections</th>
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
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.hook_resource || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.action || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.project_slug || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{sentryWebhookIssueLabel(item) || "—"}</td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">
                {sentryWebhookOutcomeLabel(item.outcome)}
              </td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.http_status}</td>
              <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">
                {item.installation_uuid || "—"}
              </td>
              <td className="px-4 py-2.5 text-gray-800 dark:text-gray-100">{item.integration_count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
