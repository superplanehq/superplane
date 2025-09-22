import React from 'react';

interface EventSourceZeroStateProps {
  eventSourceType: string;
}

const getZeroStateMessage = (eventSourceType: string): string => {
  switch (eventSourceType) {
    case 'semaphore':
      return 'Listening to changes in your Semaphore project';
    case 'github':
      return 'Listening to changes in your GitHub repository';
    case 'webhook':
      return 'Ready to receive webhook events';
    default:
      return 'Ready to receive events';
  }
};

export const EventSourceZeroState: React.FC<EventSourceZeroStateProps> = ({
  eventSourceType
}) => {
  return (
    <div className="bg-zinc-50 dark:bg-zinc-800 px-4 rounded-b-lg border-t border-gray-200 dark:border-gray-700">
      <div className="text-center py-4 pt-6 pb-4">
        <span
          className="material-symbols-outlined select-none inline-flex items-center justify-center !w-12 !h-12 !text-[48px] !leading-12 mx-auto text-zinc-400 dark:text-zinc-500 mb-2 animate-pulse"
          aria-hidden="true"
          style={{ fontVariationSettings: '"FILL" 0, "wght" 400, "GRAD" 0, "opsz" 24' }}
        >
          sensors
        </span>
        <h3 className="font-semibold text-zinc-900 dark:text-white mb-2 !text-sm text-2xl/8 font-semibold text-zinc-950 sm:text-xl/8 dark:text-white">
          Ready to receive events
        </h3>
        <p className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-xs text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">
          {getZeroStateMessage(eventSourceType)}
        </p>
      </div>
    </div>
  );
};