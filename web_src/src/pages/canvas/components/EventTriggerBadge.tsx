import React, { useMemo } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { SuperplaneStageEvent, SuperplaneExecution } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';
import Tippy from '@tippyjs/react/headless';
import { useCanvasStore } from '../store/canvasStore';

interface EventTriggerBadgeProps {
  lastExecutionEvent?: SuperplaneStageEvent;
  lastExecution?: SuperplaneExecution;
  stageName?: string;
}

export const EventTriggerBadge: React.FC<EventTriggerBadgeProps> = ({
  lastExecutionEvent,
  lastExecution,
  stageName
}) => {
  const { nodes } = useCanvasStore(state => state);

  const eventSourceNode = useMemo(() => nodes.find(node => node.id === lastExecutionEvent?.sourceId), [nodes, lastExecutionEvent?.sourceId]);

  if (!lastExecutionEvent?.id) return null;

  return (
    <Tippy
      render={attrs => (
        <div className="bg-white dark:bg-zinc-800 p-4 rounded-lg border border-gray-200 dark:border-zinc-700 max-w-sm" {...attrs}>
          <div className="text-sm font-semibold text-gray-900 dark:text-white mb-3">Event Trigger</div>
          <div className="space-y-3">
            <div className="relative">
              <div className="flex items-start gap-3">
                <div className="relative flex-shrink-0 mt-1">
                  <div className="w-2 h-2 rounded-full bg-gray-400 dark:bg-zinc-500"></div>
                  <div className="absolute top-2 left-1 w-px h-6 bg-gray-300 dark:bg-zinc-600"></div>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium truncate text-gray-900 dark:text-white">Source</span>
                    <span className="text-xs text-gray-500 dark:text-zinc-400 flex-shrink-0 ml-2">{formatRelativeTime(lastExecutionEvent?.createdAt)}</span>
                  </div>
                  <div className="text-xs text-gray-600 dark:text-zinc-400 mt-1">Incoming event trigger</div>
                  <div className="text-xs font-mono text-gray-500 dark:text-zinc-500 mt-1">{eventSourceNode?.data?.name || 'Unknown'}</div>
                </div>
              </div>
            </div>
            <div className="relative">
              <div className="flex items-start gap-3 bg-blue-50 dark:bg-blue-900/20 p-2 rounded border border-blue-200 dark:border-blue-800">
                <div className="relative flex-shrink-0 mt-1">
                  <div className="w-2 h-2 rounded-full bg-blue-500"></div>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium truncate text-blue-900 dark:text-blue-100">{stageName}</span>
                    <span className="text-xs text-gray-500 dark:text-zinc-400 flex-shrink-0 ml-2">{formatRelativeTime(lastExecution?.createdAt)}</span>
                  </div>
                  <div className="text-xs text-gray-600 dark:text-zinc-400 mt-1">Current workflow execution</div>
                  <div className="text-xs font-mono text-gray-500 dark:text-zinc-500 mt-1">current_event</div>
                </div>
              </div>
            </div>
          </div>
          <div className="mt-4 pt-3 border-t border-gray-200 dark:border-zinc-700">
            <div className="text-xs text-gray-600 dark:text-zinc-400">
              <span className="font-medium">Current trigger:</span> {eventSourceNode?.data?.name || 'Unknown'}
            </div>
            <div className="text-xs text-gray-600 dark:text-zinc-400 mt-1">
              <span className="font-medium">Event ID:</span> <span className="font-mono">{lastExecutionEvent?.id}</span>
            </div>
          </div>
        </div>
      )}
      placement="top"
    >
      <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10 max-w-28">
        <MaterialSymbol name="bolt" size="md" />
        <span className="truncate">Event {lastExecutionEvent?.id || 'N/A'}</span>
      </span>
    </Tippy>
  );
};