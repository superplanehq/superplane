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
      case 'STATE_DISCARDED':
        return 'discarded';
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
          dotColor: 'bg-yellow-500',
          textColor: 'text-yellow-700 dark:text-yellow-400',
          label: 'Pending',
          animate: true,
        };
      case 'discarded':
        return {
          dotColor: 'bg-zinc-500',
          textColor: 'text-zinc-600 dark:text-zinc-400',
          label: 'Discarded',
          animate: false,
        };
      case 'processed':
        return {
          dotColor: 'bg-green-500',
          textColor: 'text-green-600 dark:text-green-400',
          label: 'Forwarded',
          animate: false,
        };
    }
  };

  const stateConfig = getStateConfig();


  const getTruncatedUrl = () => {
    // Create a truncated version of the event ID to simulate a URL
    if (eventId.length > 30) {
      return eventId.substring(0, 27) + '...';
    }
    return eventId;
  };


  return (
    <div className="border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-lg text-left">
      <div className="p-3">
        <div className="cursor-pointer flex items-center justify-between" onClick={toggleExpand}>
          <div className="flex items-center gap-2 truncate pr-2">
            <div className={`w-2 h-2 rounded-full flex-shrink-0 ${stateConfig.dotColor} ${stateConfig.animate ? 'animate-pulse' : ''}`}></div>
            <span className="font-medium truncate text-sm dark:text-white font-mono">{getTruncatedUrl()}</span>
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
            state={state}
            eventType={eventType}
            sourceName={sourceName}
            showDetailsTab={true}
          />
        )}
      </div>
    </div>
  );
});