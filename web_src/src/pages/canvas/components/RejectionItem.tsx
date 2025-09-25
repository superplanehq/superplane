import React, { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { EventRejectionRejectionReason } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';
import { PayloadDisplay } from './PayloadDisplay';

interface RejectionItemProps {
  rejection: {
    id?: string;
    event?: {
      id?: string;
      type?: string;
      sourceName?: string;
      receivedAt?: string;
      raw?: { [key: string]: unknown };
      headers?: { [key: string]: unknown };
    };
    targetType?: string;
    targetId?: string;
    targetName?: string;
    reason?: EventRejectionRejectionReason;
    message?: string;
    rejectedAt?: string;
  };
}

export const RejectionItem: React.FC<RejectionItemProps> = React.memo(({
  rejection,
}) => {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const getRejectionReasonConfig = (reason?: EventRejectionRejectionReason) => {
    switch (reason) {
      case 'REJECTION_REASON_FILTERED':
        return {
          bgColor: 'bg-zinc-100/50 dark:bg-zinc-900/20',
          textColor: 'text-zinc-600 dark:text-zinc-400',
          icon: 'filter_alt',
          label: 'Filtered',
        };
      case 'REJECTION_REASON_ERROR':
        return {
          bgColor: 'bg-red-100/50 dark:bg-red-900/20',
          textColor: 'text-red-600 dark:text-red-400',
          icon: 'error',
          label: 'Error',
        };
      case 'REJECTION_REASON_UNKNOWN':
      default:
        return {
          bgColor: 'bg-zinc-100/50 dark:bg-zinc-900/20',
          textColor: 'text-zinc-600 dark:text-zinc-400',
          icon: 'help',
          label: 'UNKNOWN',
        };
    }
  };

  const eventId = rejection.event?.id || rejection.id || 'unknown';
  const eventType = rejection.event?.type;
  const sourceName = rejection.event?.sourceName;
  const timestamp = rejection.rejectedAt || rejection.event?.receivedAt;
  const payload = rejection.event?.raw;
  const headers = rejection.event?.headers;

  return (
    <div className="border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-lg text-left">
      <div className="p-3">
        <div className="cursor-pointer flex items-center justify-between" onClick={toggleExpand}>
          <div className="flex items-center gap-2 truncate pr-2">
            {/* Rejection reason badge */}
            <span className={`inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline ${getRejectionReasonConfig(rejection.reason).bgColor} ${getRejectionReasonConfig(rejection.reason).textColor}`}>
              <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-base" aria-hidden="true">{getRejectionReasonConfig(rejection.reason).icon}</span>
              <span>{getRejectionReasonConfig(rejection.reason).label}</span>
            </span>
            {/* Event type badge */}
            {eventType && (
              <span className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-mono bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200 border border-gray-200 dark:border-gray-700">
                <span>{eventType}</span>
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
              {timestamp ? formatRelativeTime(timestamp, true) : 'Unknown time'}
            </span>
            <MaterialSymbol
              name={isExpanded ? 'expand_less' : 'expand_more'}
              size="xl"
              className="text-gray-600 dark:text-zinc-400"
            />
          </div>
        </div>

        {isExpanded && (
          <div className="mt-4 space-y-4">
            {/* Rejection message */}
            {rejection.message && (
              <div className="bg-zinc-50 dark:bg-zinc-800/50 border border-zinc-200 dark:border-zinc-700 rounded-lg p-3">
                <div>
                  <div className="text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
                    Message
                  </div>
                  <div className="text-sm text-zinc-800 dark:text-zinc-200">
                    {rejection.message}
                  </div>
                </div>
              </div>
            )}

            {/* Event payload and headers */}
            {(payload || headers) && (
              <PayloadDisplay
                headers={headers}
                payload={payload}
                eventId={eventId}
                timestamp={timestamp}
                eventType={eventType}
                sourceName={sourceName}
                showDetailsTab={true}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
});