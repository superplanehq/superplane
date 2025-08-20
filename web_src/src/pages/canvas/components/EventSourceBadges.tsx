import React from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { SuperplaneEventSource, IntegrationsIntegration, SuperplaneFilter } from '@/api-client';
import Tippy from '@tippyjs/react/headless';

interface EventSourceBadgesProps {
  resourceName?: string;
  currentEventSource?: SuperplaneEventSource;
  eventSourceType?: string;
  integration?: IntegrationsIntegration;
}

export const EventSourceBadges: React.FC<EventSourceBadgesProps> = ({
  resourceName,
  currentEventSource,
  eventSourceType,
  integration
}) => {
  const totalFilters = currentEventSource?.spec?.events?.reduce(
    (count, event) => count + (event.filters?.length || 0),
    0
  ) || 0;

  const getResourceTypeLabel = () => {
    if (eventSourceType === 'github') return 'Repository';
    if (eventSourceType === 'semaphore') return 'Project';
    return 'Resource';
  };

  const getFilterTypeLabel = (filter: SuperplaneFilter) => {
    if (filter.type === 'FILTER_TYPE_DATA') return 'Data';
    if (filter.type === 'FILTER_TYPE_HEADER') return 'Header';
    return 'Unknown';
  };

  const getFilterExpression = (filter: SuperplaneFilter) => {
    if (filter.data?.expression) return filter.data.expression;
    if (filter.header?.expression) return filter.header.expression;
    return 'No expression';
  };

  const cleanResourceName = resourceName?.replace('.semaphore/', '') || '';

  return (
    <div className="flex items-center w-full gap-2 px-4 pb-4 font-semibold">
      {resourceName && (
        <Tippy
          render={attrs => (
            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[250px]" {...attrs}>
              <div className="text-sm font-medium text-zinc-900 dark:text-white mb-3">{getResourceTypeLabel()} Configuration</div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-zinc-600 dark:text-zinc-400">{getResourceTypeLabel()}:</span>
                  <span className="text-sm font-mono text-zinc-800 dark:text-zinc-200 bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded">{cleanResourceName}</span>
                </div>
                {integration && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-zinc-600 dark:text-zinc-400">Integration:</span>
                    <span className="text-sm font-medium text-zinc-900 dark:text-white ml-2">{integration.metadata?.name}</span>
                  </div>
                )}
              </div>
            </div>
          )}
          placement="top"
        >
          <div className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
            <MaterialSymbol name="assignment" size="md" />
            <span className="truncate">{cleanResourceName}</span>
          </div>
        </Tippy>
      )}

      {totalFilters > 0 && (
        <Tippy
          render={attrs => (
            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[280px]" {...attrs}>
              <div className="space-y-3">
                {currentEventSource?.spec?.events?.map((event, eventIndex) => (
                  <div key={eventIndex}>
                    {event.type && (
                      <div className="text-sm font-medium text-zinc-900 dark:text-white mb-2">
                        Event type: <span className="font-mono text-zinc-700 dark:text-zinc-300">{event.type}</span>
                      </div>
                    )}
                    <div className="space-y-1">
                      {event.filters?.map((filter, filterIndex) => (
                        <div key={filterIndex}>
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-1 text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded">
                              <span className="text-xs bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">{getFilterTypeLabel(filter)}</span>
                              <span className="text-zinc-500 dark:text-zinc-400">matches</span>
                              <span className="font-mono text-zinc-500 dark:text-zinc-400 ml-1">{getFilterExpression(filter)}</span>
                            </div>
                          </div>
                          {filterIndex < (event.filters?.length || 0) - 1 && (
                            <span className="mt-1 text-xs block text-center text-zinc-500 dark:text-zinc-400">{event.filterOperator === 'FILTER_OPERATOR_OR' ? 'OR' : 'AND'}</span>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
          placement="top"
        >
          <div className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
            <MaterialSymbol name="filter_list" size="md" />
            <span>{totalFilters} Event {totalFilters === 1 ? 'filter' : 'filters'}</span>
          </div>
        </Tippy>
      )}
    </div>
  );
};