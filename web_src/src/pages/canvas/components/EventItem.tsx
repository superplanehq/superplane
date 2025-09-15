import React, { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { SuperplaneEventState } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';
import { PayloadDisplay } from './PayloadDisplay';

interface EventItemProps {
  eventId: string;
  timestamp: string;
  state?: SuperplaneEventState;
  eventType?: string;
  sourceName?: string;
  headers?: { [key: string]: unknown };
  payload?: { [key: string]: unknown };
}

export const EventItem: React.FC<EventItemProps> = React.memo(({
  eventId,
  timestamp,
  state,
  eventType,
  sourceName,
  headers,
  payload,
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
          label: 'Forwarded',
          animate: false,
        };
    }
  };

  const stateConfig = getStateConfig();


  return (
    <div className="border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-lg text-left">
      <div className="p-3">
        <div className="cursor-pointer flex items-center justify-between" onClick={toggleExpand}>
          <div className="flex items-center gap-2 truncate pr-2">
            {/* State badge */}
            <span className={`inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline ${stateConfig.bgColor} ${stateConfig.textColor}`}>
              <span className={`material-symbols-outlined select-none inline-flex items-center justify-center !text-base ${stateConfig.animate ? 'animate-pulse' : ''}`} aria-hidden="true">{stateConfig.icon}</span>
              <span className="uppercase">{stateConfig.label}</span>
            </span>
            {/* Event type badge */}
            <span className="inline-flex items-center rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-blue-100/50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400">
              <span>{eventType || 'webhook'}</span>
            </span>
          </div>
          <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap text-right">{formatRelativeTime(timestamp, true)}</span>
          <div className="flex items-center gap-3">
            <MaterialSymbol
              name={isExpanded ? 'expand_less' : 'expand_more'}
              size="xl"
              className="text-gray-600 dark:text-zinc-400"
            />
          </div>
        </div>

        {isExpanded && (
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
    </div>
  );
});