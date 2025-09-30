import React, { useMemo, useCallback } from 'react';
import { Badge } from '@/components/Badge/badge';
import { SuperplaneEventSource, IntegrationsIntegration, SuperplaneFilter } from '@/api-client';
import Tippy from '@tippyjs/react/headless';
import { getResourceLabel } from '@/utils/components';

interface EventSourceBadgesProps {
  resourceName?: string;
  currentEventSource?: SuperplaneEventSource;
  sourceType: string;
  integration?: IntegrationsIntegration;
}

export const EventSourceBadges: React.FC<EventSourceBadgesProps> = ({
  resourceName,
  currentEventSource,
  sourceType,
  integration
}) => {
  const totalFilters = currentEventSource?.spec?.events?.reduce(
    (count, event) => count + (event.filters?.length || 0),
    0
  ) || 0;

  const totalEventTypes = currentEventSource?.spec?.events?.length || 0;

  const getFilterTypeLabel = useCallback((filter: SuperplaneFilter) => {
    if (filter.type === 'FILTER_TYPE_DATA') return 'Data';
    if (filter.type === 'FILTER_TYPE_HEADER') return 'Header';
    return 'Unknown';
  }, []);

  const getFilterExpression = useCallback((filter: SuperplaneFilter) => {
    if (filter.data?.expression) return filter.data.expression;
    if (filter.header?.expression) return filter.header.expression;
    return 'No expression';
  }, []);

  const cleanResourceName = resourceName?.replace('.semaphore/', '') || '';

  const resourceUrl = currentEventSource?.spec?.resource?.url;

  const badgeItems = useMemo(() => {
    const badges: Array<{ icon: string; text: string; tooltip: React.ReactNode; url?: string }> = [];

    if (resourceName) {
      badges.push({
        icon: 'assignment',
        text: cleanResourceName,
        url: resourceUrl,
        tooltip: (
          <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[250px]">
            <div className="text-sm font-medium text-zinc-900 dark:text-white mb-3">{getResourceLabel(sourceType)} Configuration</div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-zinc-600 dark:text-zinc-400">{getResourceLabel(sourceType)}:</span>
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
        )
      });
    }

    if (totalFilters > 0 || totalEventTypes > 0) {
      const filterText = totalFilters > 0
        ? `${totalFilters} Event ${totalFilters === 1 ? 'filter' : 'filters'}`
        : `${totalEventTypes} Event ${totalEventTypes === 1 ? 'type' : 'types'}`;

      badges.push({
        icon: 'filter_list',
        text: filterText,
        tooltip: (
          <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[280px]">
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
        )
      });
    }

    return badges;
  }, [resourceName, cleanResourceName, totalFilters, totalEventTypes, currentEventSource, integration, getFilterTypeLabel, getFilterExpression, resourceUrl]);

  if (badgeItems.length === 0) return null;

  return (
    <div className="flex items-center w-full gap-2 px-4 pb-4 font-semibold min-w-0 overflow-hidden">
      {badgeItems.map((badge, index) => (
        <Tippy
          key={`${badge.icon}-${index}`}
          render={attrs => <div {...attrs}>{badge.tooltip}</div>}
          placement="top"
        >
          <div className="flex-shrink min-w-0 max-w-full">
            {badge.url ? (
              <a href={badge.url} target="_blank" rel="noopener noreferrer" className="inline-block">
                <Badge
                  color="zinc"
                  icon={badge.icon}
                  truncate
                >
                  {badge.text}
                </Badge>
              </a>
            ) : (
              <Badge
                color="zinc"
                icon={badge.icon}
                truncate
              >
                {badge.text}
              </Badge>
            )}
          </div>
        </Tippy>
      ))}
    </div>
  );
};