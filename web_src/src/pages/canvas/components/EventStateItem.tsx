import React from 'react';
import { formatRelativeTime } from '../utils/stageEventUtils';

export type EventState = 'pending' | 'discarded' | 'processed';

interface EventStateItemProps {
  eventId: string;
  state: EventState;
  receivedAt?: string;
  onClick?: () => void;
}

export const EventStateItem: React.FC<EventStateItemProps> = ({
  eventId,
  state,
  receivedAt,
  onClick
}) => {
  const getStateConfig = (state: EventState) => {
    switch (state) {
      case 'pending':
        return {
          dotColor: 'bg-yellow-500',
          textColor: 'text-yellow-600 dark:text-yellow-400',
          label: 'Pending',
          animate: true,
          strikeThrough: false
        };
      case 'discarded':
        return {
          dotColor: 'bg-zinc-500',
          textColor: 'text-zinc-600 dark:text-zinc-400',
          label: 'Discarded',
          animate: false,
          strikeThrough: true
        };
      case 'processed':
        return {
          dotColor: 'bg-green-500',
          textColor: 'text-green-600 dark:text-green-400',
          label: 'Forwarded',
          animate: false,
          strikeThrough: false
        };
    }
  };

  const config = getStateConfig(state);
  const timeAgo = receivedAt ? formatRelativeTime(receivedAt, true) : '';

  return (
    <div
      className="flex items-center gap-2 p-2 bg-gray-50 dark:bg-zinc-800 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors duration-150 cursor-pointer"
      onClick={onClick}
    >
      <div className="flex items-center gap-2">
        <div className={`w-2 h-2 ${config.dotColor} rounded-full flex-shrink-0 ${config.animate ? 'animate-pulse' : ''}`}></div>
        <span className={`text-xs font-semibold ${config.textColor}`}>{config.label}</span>
      </div>
      <span className={`text-sm font-mono text-gray-800 dark:text-zinc-200 truncate flex-1 ${config.strikeThrough ? 'line-through opacity-60' : ''}`}>
        {eventId}
      </span>
      {timeAgo && (
        <span className="text-xs text-zinc-500 dark:text-zinc-400 flex-shrink-0 w-6 text-right">
          {timeAgo.replace(' ago', '')}
        </span>
      )}
    </div>
  );
};