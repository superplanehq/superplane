import React from 'react';
import { SuperplaneStageEvent } from '@/api-client';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface ApprovalQueueItemProps {
  event: SuperplaneStageEvent;
  onApprove: (eventId: string, stageId: string) => void;
  stageId: string;
}

export const ApprovalQueueItem: React.FC<ApprovalQueueItemProps> = ({ event, onApprove, stageId }) => {
  return (
    <div className="space-y-2 mb-2">
      <div className="flex items-center p-2 border bg-orange-50 dark:bg-orange-900/20 border-orange-400 dark:border-orange-800 rounded-sm gap-2 justify-between">
        <div className="flex items-center gap-2 truncate">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full flex-shrink-0 bg-orange-600 dark:bg-orange-500 animate-pulse"></div>
            <span className="text-xs font-medium text-orange-700 dark:text-orange-500">Action required</span>
          </div>
          <span className="text-sm text-gray-700 dark:text-gray-200 truncate font-medium">{event.label || event.id}</span>
        </div>
        <div className="flex items-center gap-1">
          {/*
          TODO: Add dismiss button when this feature is implemented
          <button
            className="relative isolate inline-flex items-baseline justify-center gap-x-2 rounded-lg border text-base/6 font-semibold px-[calc(theme(spacing[3.5])-1px)] py-[calc(theme(spacing[2.5])-1px)] sm:px-[calc(theme(spacing[3])-1px)] sm:py-[calc(theme(spacing[1.5])-1px)] sm:text-sm/6 focus:outline-2 focus:outline-offset-2 focus:outline-blue-500 border-transparent text-zinc-950 hover:bg-zinc-950/5 dark:text-white dark:hover:bg-white/10 cursor-pointer"
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              // Handle close/dismiss action
            }}
          >
            <MaterialSymbol name="close" size="sm" className="text-gray-700 dark:text-gray-400" />
          </button>
          */}
          <button
            className="relative isolate inline-flex items-baseline justify-center gap-x-2 rounded-lg border text-base/6 font-semibold px-[calc(theme(spacing[3.5])-1px)] py-[calc(theme(spacing[2.5])-1px)] sm:px-[calc(theme(spacing[3])-1px)] sm:py-[calc(theme(spacing[1.5])-1px)] sm:text-sm/6 focus:outline-2 focus:outline-offset-2 focus:outline-blue-500 border-transparent bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700 cursor-pointer shadow-sm"
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              if (event.id) {
                onApprove(event.id, stageId);
              }
            }}
          >
            <MaterialSymbol name="check" size="sm" className="text-gray-700 dark:text-gray-400" />
          </button>
        </div>
      </div>
    </div>
  );
};

interface WaitingQueueItemProps {
  event: SuperplaneStageEvent;
  label?: string;
}

export const WaitingQueueItem: React.FC<WaitingQueueItemProps> = ({ event, label = "Waiting" }) => {
  return (
    <div className="space-y-2 mb-2">
      <div className="flex items-center p-2 border bg-zinc-50 dark:bg-zinc-700 border-gray-200 dark:border-gray-700 rounded-md gap-2 justify-between">
        <div className="flex items-center gap-2 truncate">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full flex-shrink-0 bg-orange-600 dark:bg-orange-500 animate-pulse"></div>
            <span className="text-xs font-medium text-orange-700 dark:text-orange-500">{label}</span>
          </div>
          <span className="text-sm text-gray-700 dark:text-gray-200 truncate font-medium">{event.label || event.id}</span>
        </div>
        <div className="flex items-center">
          <MaterialSymbol name="timer" size="lg" className="text-orange-700 dark:text-orange-600 px-2" />
        </div>
      </div>
    </div>
  );
};

interface PendingQueueItemProps {
  event: SuperplaneStageEvent;
}

export const PendingQueueItem: React.FC<PendingQueueItemProps> = ({ event }) => {
  return (
    <div className="space-y-2">
      <div className="flex items-center p-2 border bg-zinc-50 dark:bg-zinc-700 border-gray-200 dark:border-gray-700 rounded-md gap-2 justify-between">
        <div className="flex items-center gap-2 truncate">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full flex-shrink-0 bg-orange-600 dark:bg-orange-500 animate-pulse"></div>
            <span className="text-xs font-medium text-orange-700 dark:text-orange-500">Pending</span>
          </div>
          <span className="text-sm text-gray-700 dark:text-gray-200 truncate font-medium">{event.label || event.id}</span>
        </div>
        <div className="flex items-center">
          <MaterialSymbol name="timer" size="lg" className="text-orange-700 dark:text-orange-600 px-2" />
        </div>
      </div>
    </div>
  );
};

export const EmptyQueueItem: React.FC = () => {
  return (
    <div className="flex justify-between w-full mb-2 px-2 py-3 border-1 rounded border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
      <span className='font-semibold text-gray-500 dark:text-gray-400 text-sm truncate mt-[2px]'>No events in queue..</span>
    </div>
  );
};