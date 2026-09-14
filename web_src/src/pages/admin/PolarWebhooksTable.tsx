import { Text } from "@/components/Text/text";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { ChevronDown, ChevronRight } from "lucide-react";
import React from "react";

import {
  formatPolarPayload,
  polarWebhookEventStatus,
  polarWebhookEventStatusLabel,
  POLAR_WEBHOOKS_REDELIVER,
  POLAR_WEBHOOKS_SENDING_AGAIN,
  type PolarWebhookDelivery,
  type PolarWebhookEventStatus,
} from "./polarWebhookDeliveries";

const deliveryBadgeClass = (succeeded: boolean) =>
  succeeded
    ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300"
    : "bg-red-100 text-red-800 dark:bg-red-950/40 dark:text-red-300";

const eventStatusBadgeClass = (status: PolarWebhookEventStatus) => {
  if (status === "sending") {
    return "bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300";
  }
  if (status === "succeeded") {
    return "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300";
  }
  return "bg-red-100 text-red-800 dark:bg-red-950/40 dark:text-red-300";
};

export function PolarWebhooksTable({
  items,
  expandedIds,
  redelivering,
  onToggle,
  onRedeliver,
}: {
  items: PolarWebhookDelivery[];
  expandedIds: Set<string>;
  redelivering: Set<string>;
  onToggle: (id: string) => void;
  onRedeliver: (eventId: string) => void;
}) {
  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-100 dark:border-gray-700/70">
            <th className="w-8 px-2 py-2.5"></th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Time</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Event type</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">HTTP status</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Result</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Event status</th>
            <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Event ID</th>
            <th className="text-right px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Action</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <PolarWebhookRow
              key={item.id}
              item={item}
              expanded={expandedIds.has(item.id)}
              pendingEventIds={redelivering}
              onToggle={onToggle}
              onRedeliver={onRedeliver}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PolarWebhookRow({
  item,
  expanded,
  pendingEventIds,
  onToggle,
  onRedeliver,
}: {
  item: PolarWebhookDelivery;
  expanded: boolean;
  pendingEventIds: ReadonlySet<string>;
  onToggle: (id: string) => void;
  onRedeliver: (eventId: string) => void;
}) {
  const eventStatus = polarWebhookEventStatus(item, pendingEventIds);
  const redelivering = pendingEventIds.has(item.event_id);

  return (
    <React.Fragment>
      <tr className="border-b border-slate-50 last:border-0 hover:bg-slate-50 transition-colors dark:border-gray-800/70 dark:hover:bg-gray-800/50">
        <td className="px-2 py-2.5">
          <button
            type="button"
            className="rounded p-1 text-gray-500 hover:bg-slate-100 dark:hover:bg-gray-800"
            aria-expanded={expanded}
            aria-label={expanded ? "Hide delivery details" : "Show delivery details"}
            onClick={() => onToggle(item.id)}
          >
            {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          </button>
        </td>
        <td className="px-4 py-2.5">
          <Timestamp
            date={item.created_at}
            display="relative"
            className="text-gray-600 whitespace-nowrap dark:text-gray-400"
            fallback={<span className="text-gray-400 dark:text-gray-500">—</span>}
          />
        </td>
        <td className="px-4 py-2.5 font-mono text-xs text-gray-800 dark:text-gray-100">{item.event_type || "—"}</td>
        <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">{item.http_code ?? "—"}</td>
        <td className="px-4 py-2.5">
          <span
            className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${deliveryBadgeClass(item.succeeded)}`}
          >
            {item.succeeded ? "Succeeded" : "Failed"}
          </span>
        </td>
        <td className="px-4 py-2.5">
          <span
            data-testid="polar-webhook-event-status"
            className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${eventStatusBadgeClass(eventStatus)}`}
          >
            {polarWebhookEventStatusLabel(eventStatus)}
          </span>
        </td>
        <td className="px-4 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300" title={item.event_id}>
          {item.event_id || "—"}
        </td>
        <td className="px-4 py-2.5 text-right">
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={!item.event_id || redelivering}
            onClick={() => onRedeliver(item.event_id)}
          >
            {redelivering ? POLAR_WEBHOOKS_SENDING_AGAIN : POLAR_WEBHOOKS_REDELIVER}
          </Button>
        </td>
      </tr>
      {expanded ? (
        <tr className="border-b border-slate-50 bg-slate-50/70 dark:border-gray-800/70 dark:bg-gray-950/40">
          <td colSpan={8} className="px-6 py-3">
            <div className="grid gap-3 md:grid-cols-2">
              <PolarDetailBlock label="Polar response" value={item.response} />
              <PolarDetailBlock label="Event payload" value={formatPolarPayload(item.payload)} />
            </div>
          </td>
        </tr>
      ) : null}
    </React.Fragment>
  );
}

function PolarDetailBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <Text className="mb-1 text-xs font-medium text-gray-500 dark:text-gray-400">{label}</Text>
      <pre className="max-h-64 overflow-auto rounded-md bg-white p-3 text-xs text-gray-800 outline outline-slate-950/10 dark:bg-gray-900 dark:text-gray-100 dark:outline-gray-700/70">
        {value.trim() === "" ? "—" : value}
      </pre>
    </div>
  );
}
