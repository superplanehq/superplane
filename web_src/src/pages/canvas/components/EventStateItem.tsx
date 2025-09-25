import React from 'react';
import { formatFullTimestamp, formatRelativeTime } from '../utils/stageEventUtils';
import Tippy from '@tippyjs/react/headless';
import 'tippy.js/dist/tippy.css';
import { SuperplaneEventState } from '@/api-client';

interface EventStateItemProps {
  state?: SuperplaneEventState;
  receivedAt?: string;
  eventType?: string;
  onClick?: () => void;
}

export const EventStateItem: React.FC<EventStateItemProps> = ({
  state,
  receivedAt,
  eventType,
  onClick
}) => {
  const getStateConfig = (state: SuperplaneEventState) => {
    switch (state) {
      case 'STATE_PENDING':
        return {
          bgColor: 'bg-yellow-100/50 dark:bg-yellow-900/20',
          textColor: 'text-yellow-700 dark:text-yellow-400',
          icon: 'schedule',
          label: 'Pending',
          animate: true,
        };
      case 'STATE_REJECTED':
        return {
          bgColor: 'bg-zinc-100/50 dark:bg-zinc-900/20',
          textColor: 'text-zinc-600 dark:text-zinc-400',
          icon: 'cancel',
          label: 'Rejected',
          animate: false,
        };
      case 'STATE_PROCESSED':
        return {
          bgColor: 'bg-green-100/50 dark:bg-green-900/20',
          textColor: 'text-green-600 dark:text-green-400',
          icon: 'check_circle',
          label: 'Processed',
          animate: false,
        };
      case 'STATE_UNKNOWN':
      default:
        return {
          bgColor: 'bg-gray-100/50 dark:bg-gray-900/20',
          textColor: 'text-gray-600 dark:text-gray-400',
          icon: 'help',
          label: 'Unknown',
          animate: false,
        };
    }
  };

  const config = getStateConfig(state || 'STATE_UNKNOWN');
  const timeAgo = receivedAt ? formatRelativeTime(receivedAt, true) : '';

  return (
    <div className="bg-zinc-50 dark:bg-zinc-800 rounded-lg text-left">
      <div className="p-1">
        <div className="cursor-pointer flex items-center justify-between" onClick={onClick}>
          <div className="flex items-center gap-2 truncate pr-2">
            {/* State badge */}
            <span className={`inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline ${config.bgColor} ${config.textColor}`}>
              <span className={`material-symbols-outlined select-none inline-flex items-center justify-center !text-base ${config.animate ? 'animate-pulse' : ''}`} aria-hidden="true">{config.icon}</span>
              <span>{config.label}</span>
            </span>
            {/* Event type badge */}
            {eventType && (
              <span className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-mono bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200 border border-gray-200 dark:border-gray-700">
                <span>{eventType}</span>
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            {timeAgo && (
              <Tippy
                render={attrs => (
                  <div className="bg-white text-center dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-3" {...attrs}>
                    {formatFullTimestamp(receivedAt)}
                  </div>
                )}
                placement="top"
              >
                <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                  {formatRelativeTime(receivedAt!, true)}
                </span>
              </Tippy>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};