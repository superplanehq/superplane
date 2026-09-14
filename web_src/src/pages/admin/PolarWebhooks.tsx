import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Webhook } from "lucide-react";
import { useState } from "react";

import AdminPagination from "./AdminPagination";
import { PolarWebhooksTable } from "./PolarWebhooksTable";
import {
  POLAR_WEBHOOK_ALL_VALUE,
  POLAR_WEBHOOK_EVENT_TYPES,
  POLAR_WEBHOOK_PAGE_SIZE,
  POLAR_WEBHOOKS_EMPTY,
  POLAR_WEBHOOKS_HELP,
  POLAR_WEBHOOKS_NOT_CONFIGURED,
  POLAR_WEBHOOKS_REDELIVER_FAILED,
  POLAR_WEBHOOKS_TITLE,
  type PolarWebhookStatusFilter,
} from "./polarWebhookDeliveries";
import { usePolarWebhooks } from "./usePolarWebhooks";

export function PolarWebhooks() {
  const pageState = usePolarWebhooks();
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  useReportPageReady(!pageState.loading || pageState.configured !== null);

  const toggleExpanded = (id: string) => {
    setExpandedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  if (pageState.loading && pageState.configured === null) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading Polar webhook deliveries...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PolarWebhooksHeader
        showBulk={pageState.configured === true && pageState.failedEventIds.length > 0}
        bulkBusy={pageState.bulkBusy}
        onRedeliverFailed={() => void pageState.handleRedeliverFailed()}
      />
      <PolarWebhooksBody pageState={pageState} expandedIds={expandedIds} onToggle={toggleExpanded} />
    </div>
  );
}

function PolarWebhooksHeader({
  showBulk,
  bulkBusy,
  onRedeliverFailed,
}: {
  showBulk: boolean;
  bulkBusy: boolean;
  onRedeliverFailed: () => void;
}) {
  return (
    <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{POLAR_WEBHOOKS_TITLE}</h1>
        <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">{POLAR_WEBHOOKS_HELP}</Text>
      </div>
      {showBulk ? (
        <Button
          type="button"
          variant="outline"
          onClick={onRedeliverFailed}
          disabled={bulkBusy}
          data-testid="polar-webhooks-redeliver-failed"
        >
          {POLAR_WEBHOOKS_REDELIVER_FAILED}
        </Button>
      ) : null}
    </div>
  );
}

function PolarWebhooksBody({
  pageState,
  expandedIds,
  onToggle,
}: {
  pageState: ReturnType<typeof usePolarWebhooks>;
  expandedIds: Set<string>;
  onToggle: (id: string) => void;
}) {
  if (pageState.loadError && pageState.configured !== true) {
    return <PolarWebhooksNotice message={pageState.loadError} dashed />;
  }
  if (!pageState.configured) {
    return <PolarWebhooksNotice message={POLAR_WEBHOOKS_NOT_CONFIGURED} dashed />;
  }

  return (
    <>
      <PolarWebhooksFilters
        statusFilter={pageState.statusFilter}
        eventType={pageState.eventType}
        onStatusChange={pageState.changeStatusFilter}
        onEventTypeChange={pageState.changeEventType}
      />
      {pageState.items.length === 0 ? (
        <PolarWebhooksNotice message={POLAR_WEBHOOKS_EMPTY} dashed={false} />
      ) : (
        <>
          <PolarWebhooksTable
            items={pageState.items}
            expandedIds={expandedIds}
            redelivering={pageState.redelivering}
            onToggle={onToggle}
            onRedeliver={(eventId) => void pageState.handleRedeliver(eventId)}
          />
          <AdminPagination
            offset={(pageState.page - 1) * POLAR_WEBHOOK_PAGE_SIZE}
            total={pageState.total}
            pageSize={POLAR_WEBHOOK_PAGE_SIZE}
            onPageChange={(offset) => pageState.setPage(Math.floor(offset / POLAR_WEBHOOK_PAGE_SIZE) + 1)}
          />
        </>
      )}
    </>
  );
}

function PolarWebhooksFilters({
  statusFilter,
  eventType,
  onStatusChange,
  onEventTypeChange,
}: {
  statusFilter: PolarWebhookStatusFilter;
  eventType: string;
  onStatusChange: (value: PolarWebhookStatusFilter) => void;
  onEventTypeChange: (value: string) => void;
}) {
  return (
    <div className="flex flex-wrap items-end gap-4">
      <div>
        <Label htmlFor="polar-webhooks-status" className="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">
          Status
        </Label>
        <Select value={statusFilter} onValueChange={(value) => onStatusChange(value as PolarWebhookStatusFilter)}>
          <SelectTrigger id="polar-webhooks-status" data-testid="polar-webhooks-status" className="h-9 w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="succeeded">Succeeded</SelectItem>
            <SelectItem value="all">All</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div>
        <Label htmlFor="polar-webhooks-event-type" className="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">
          Event type
        </Label>
        <Select value={eventType} onValueChange={onEventTypeChange}>
          <SelectTrigger id="polar-webhooks-event-type" data-testid="polar-webhooks-event-type" className="h-9 w-56">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={POLAR_WEBHOOK_ALL_VALUE}>All</SelectItem>
            {POLAR_WEBHOOK_EVENT_TYPES.map((type) => (
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

function PolarWebhooksNotice({ message, dashed }: { message: string; dashed: boolean }) {
  const border = dashed
    ? "border-dashed border-slate-300 dark:border-gray-700"
    : "border-slate-200 dark:border-gray-700";
  return (
    <div className={`rounded-xl border bg-white p-8 text-center shadow-sm dark:bg-gray-900 ${border}`}>
      <Webhook size={24} className="mx-auto text-gray-400 dark:text-gray-500" />
      <Text className="mt-3 text-sm text-gray-600 dark:text-gray-400">{message}</Text>
    </div>
  );
}
