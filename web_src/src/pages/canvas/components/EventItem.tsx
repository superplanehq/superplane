import React, { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { SuperplaneEventState, SuperplaneEventStateReason } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';
import { PayloadDisplay } from './PayloadDisplay';

interface EventItemProps {
  eventId: string;
  timestamp: string;
  state?: SuperplaneEventState;
  stateReason?: SuperplaneEventStateReason;
  stateMessage?: string;
  eventType?: string;
  sourceName?: string;
  headers?: { [key: string]: unknown };
  payload?: { [key: string]: unknown };
  showStateLabel?: boolean;
}

export const EventItem: React.FC<EventItemProps> = React.memo(({
  eventId,
  timestamp,
  state,
  stateReason,
  stateMessage,
  eventType,
  sourceName,
  headers,
  payload,
  showStateLabel = true,
}) => {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  // Map SuperplaneEventState to EventStateItem format for the header display
  const getEventStateType = () => {
    switch (state) {
      case 'STATE_PROCESSED':
        return 'processed';
      case 'STATE_PENDING':
        return 'pending';
      case 'STATE_REJECTED':
        return 'rejected';
      default:
        return 'pending';
    }
  };

  // Use EventStateItem logic for state configuration
  const getStateConfig = () => {
    const stateType = getEventStateType();
    switch (stateType) {
      case 'pending':
        return {
          bgColor: 'bg-yellow-100/50',
          textColor: 'text-yellow-700 dark:text-yellow-400',
          icon: 'schedule',
          label: 'Pending',
          animate: true,
        };
      case 'rejected':
        return {
          bgColor: 'bg-zinc-100/50 dark:bg-zinc-900/20',
          textColor: 'text-zinc-600 dark:text-zinc-400',
          icon: 'cancel',
          label: 'Rejected',
          animate: false,
        };
      case 'processed':
        return {
          bgColor: 'bg-green-100/50 dark:bg-green-900/20',
          textColor: 'text-green-600 dark:text-green-400',
          icon: 'check_circle',
          label: 'Processed',
          animate: false,
        };
    }
  };

  const stateConfig = getStateConfig();

  // Format state reason for display
  const formatStateReason = (reason?: SuperplaneEventStateReason) => {
    switch (reason) {
      case 'STATE_REASON_FILTERED':
        return 'Filtered';
      case 'STATE_REASON_ERROR':
        return 'Error';
      case 'STATE_REASON_OK':
        return 'OK';
      default:
        return 'Unknown';
    }
  };

  return (
    <div className="border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-lg text-left">
      <div className="p-3">
        <div className="cursor-pointer flex items-center justify-between" onClick={toggleExpand}>
          <div className="flex items-center gap-2 truncate pr-2">
            {/* State badge */}
            {showStateLabel && (
              <span className={`inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline ${stateConfig.bgColor} ${stateConfig.textColor}`}>
                <span className={`material-symbols-outlined select-none inline-flex items-center justify-center !text-base ${stateConfig.animate ? 'animate-pulse' : ''}`} aria-hidden="true">{stateConfig.icon}</span>
                <span>{stateConfig.label}</span>
              </span>
            )}
            {/* Event type badge */}
            <span className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-mono bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200 border border-gray-200 dark:border-gray-700">
              <span>{eventType || 'webhook'}</span>
            </span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">{formatRelativeTime(timestamp, true)}</span>
            <MaterialSymbol
              name={isExpanded ? 'expand_less' : 'expand_more'}
              size="xl"
              className="text-gray-600 dark:text-zinc-400"
            />
          </div>
        </div>

        {isExpanded && (
          <div className="mt-4 space-y-4">
            {/* Show rejection details for rejected events */}
            {state === 'STATE_REJECTED' && (stateReason || stateMessage) && (
              <div className="bg-zinc-50 dark:bg-zinc-800/50 border border-zinc-200 dark:border-zinc-700 rounded-lg p-3">
                <div className="space-y-2">
                  {stateReason && (
                    <div>
                      <div className="text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
                        Reason
                      </div>
                      <div className="text-sm text-zinc-800 dark:text-zinc-200">
                        {formatStateReason(stateReason)}
                      </div>
                    </div>
                  )}
                  {stateMessage && (
                    <div>
                      <div className="text-xs font-medium text-zinc-600 dark:text-zinc-400 mb-1">
                        Message
                      </div>
                      <div className="text-sm text-zinc-800 dark:text-zinc-200">
                        {stateMessage}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
            <PayloadDisplay
              headers={headers}
              payload={payload}
              eventId={eventId}
              timestamp={timestamp}
              eventType={eventType}
              sourceName={sourceName}
              showDetailsTab={true}
            />
          </div>
        )}
      </div>
    </div>
  );
});