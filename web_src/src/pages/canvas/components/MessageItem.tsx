import React, { useCallback, useMemo, useRef, useState } from 'react';
import { SuperplaneStageEvent, SuperplaneStage } from '@/api-client';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { formatRelativeTime } from '../utils/stageEventUtils';

interface MessageItemProps {
  event: SuperplaneStageEvent;
  selectedStage: SuperplaneStage;
  onApprove?: (eventId: string) => void;
  onRemove?: (eventId: string) => void;
  executionRunning?: boolean;
}

const MessageItem = React.memo(({
  event,
  selectedStage,
  onApprove,
  onRemove,
}: MessageItemProps) => {
  const [isExpanded, setIsExpanded] = React.useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  const handleRemove = () => {
    if (onRemove && event.id) {
      onRemove(event.id);
    }
    setIsDropdownOpen(false);
  };

  const handleApprove = () => {
    if (onApprove && event.id) {
      onApprove(event.id);
    }
  };

  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);


  const mapEventInputs = useCallback(() => {
    const map: Record<string, string> = {};
    const eventInputs = event.inputs?.map(input => [input.name, input.value]).reduce((acc, [key, value]) => {
      acc[key!] = value!;
      return acc;
    }, {} as Record<string, string>);

    selectedStage?.spec?.inputs?.forEach((input) => {
      if (!input.name) {
        return;
      }

      map[input.name!] = eventInputs?.[input.name!] || "-";
    });

    return map;
  }, [event, selectedStage]);

  const inputsRecord = useMemo(() => mapEventInputs(), [mapEventInputs]);
  const requiredApprovals = useMemo(() =>
    selectedStage.spec?.conditions
      ?.find(condition => condition.type === "CONDITION_TYPE_APPROVAL")
      ?.approval?.count || 0,
    [selectedStage]);

  const timeWindowCondition = useMemo(() =>
    selectedStage.spec?.conditions
      ?.find(condition => condition.type === "CONDITION_TYPE_TIME_WINDOW"),
    [selectedStage]);

  const getStatusLabel = useCallback(() => {
    if (event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_APPROVAL') {
      return 'Waiting';
    } else if (event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_TIME_WINDOW') {
      return 'Waiting';
    } else if (event.state === 'STATE_PENDING') {
      return 'Pending';
    }
    return 'Pending';
  }, [event.state, event.stateReason]);

  const formatTimeWindow = useCallback(() => {
    if (timeWindowCondition?.timeWindow?.start && timeWindowCondition?.timeWindow?.end) {
      // Simple formatting - you can enhance this based on your needs
      return `Run between ${timeWindowCondition.timeWindow.start} and ${timeWindowCondition.timeWindow.end} on ${timeWindowCondition.timeWindow.weekDays?.join(', ')}`;
    }
    return 'Run at scheduled time';
  }, [timeWindowCondition]);

  const getRelativeTime = useCallback(() => {
    if (!event.createdAt) return 'now';
    return formatRelativeTime(event.createdAt, true);
  }, [event.createdAt]);

  return (
    <div className="queueItem">
      <div className="p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-pointer" onClick={toggleExpand}>
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 truncate">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full flex-shrink-0 bg-amber-600 dark:bg-amber-500 animate-pulse"></div>
              <span className="text-xs font-medium text-amber-700 dark:text-amber-500">{getStatusLabel()}</span>
            </div>
            <span className="font-medium truncate text-sm dark:text-white">
              {event.id || 'Unknown'}
            </span>
          </div>
          <div className="flex items-center gap-3">
            {!isExpanded && (
              <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                {getRelativeTime()}
              </span>
            )}
            <MaterialSymbol
              name={isExpanded ? 'expand_less' : 'expand_more'}
              size="xl"
              className="text-gray-600 dark:text-zinc-400"
            />
          </div>
        </div>

        {isExpanded && (
          <div className="mt-3 space-y-3">
            <div className="mt-3 space-y-3">
              {Object.keys(inputsRecord).length > 0 ? (
                <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-zinc-50 dark:bg-zinc-800">
                  <div className="flex items-start gap-3">
                    <div className="flex-1">
                      <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Inputs</div>
                      <div className="space-y-1">
                        {Object.entries(inputsRecord).map(([key, value]) => (
                          <div key={key} className="flex items-center justify-between gap-2 min-w-0">
                            <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium font-mono truncate">{key}</span>
                            <div className="flex items-center gap-2 flex-shrink-0">
                              <span className="font-mono !text-xs truncate inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10 max-w-32">
                                {value}
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-zinc-50 dark:bg-zinc-800">
                  <div className="flex items-start gap-3">
                    <div className="flex-1">
                      <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Inputs</div>
                      <div className="space-y-1">
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">No inputs</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Approval Footer */}
      {event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_APPROVAL' && (
        <div className="px-3 py-2 border bg-orange-50 dark:bg-orange-900/20 border-orange-400 dark:border-orange-700">
          <div className="flex justify-between items-center">
            <div className="flex items-center">
              <MaterialSymbol name="how_to_reg" size="md" className="text-orange-700 dark:text-orange-200 mr-2" />
              <span className="text-xs text-gray-700 dark:text-zinc-400">
                {event.approvals && event.approvals.length > 0 ? (
                  <>
                    <a href="#" className="black underline">
                      {event.approvals.length} person{event.approvals.length !== 1 ? 's' : ''}
                    </a>
                    {' approved'}
                    {event.approvals.length < requiredApprovals && ', ' + (requiredApprovals - event.approvals.length) + ' more needed'}
                  </>
                ) : (
                  'Waiting for approval'
                )}
              </span>
            </div>
            <div className="flex items-center">

              {/* Approve Button */}
              {onApprove && (
                <button
                  className="relative isolate inline-flex items-baseline justify-center gap-x-2 rounded-lg border text-base/6 font-semibold px-[calc(--spacing(3.5)-1px)] py-[calc(--spacing(2.5)-1px)] sm:px-[calc(--spacing(3)-1px)] sm:py-[calc(--spacing(1.5)-1px)] sm:text-sm/6 focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-blue-500 data-disabled:opacity-50 border-transparent bg-white dark:bg-zinc-800 before:absolute before:inset-0 before:-z-10 before:rounded-[calc(var(--radius-lg)-1px)] before:bg-white before:shadow-sm dark:before:hidden dark:border-white/5 after:absolute after:inset-0 after:-z-10 after:rounded-[calc(var(--radius-lg)-1px)] after:shadow-[inset_0_1px_theme(colors.white/15%)] text-zinc-950 cursor-default"
                  type="button"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleApprove();
                  }}
                >
                  <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
                  <MaterialSymbol name="check" size="sm" className="text-black-700 dark:text-black-400" />
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Time Window Footer */}
      {event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_TIME_WINDOW' && (
        <div className="px-3 py-2 border border-t-0 bg-orange-50 dark:bg-orange-900/20 border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center">
            <MaterialSymbol name="schedule" size="md" className="text-orange-700 dark:text-orange-200 mr-2" />
            <span className="text-xs text-gray-700 dark:text-zinc-400">{formatTimeWindow()}</span>
          </div>
        </div>
      )}

      {isDropdownOpen && (
        <div
          className="absolute right-0 bg-white dark:bg-zinc-800 shadow-lg rounded-lg w-32 z-10 border border-gray-200 dark:border-zinc-700"
          style={{ marginTop: '4px' }}
          ref={dropdownRef}
        >
          <div className="py-1">
            <button
              onClick={handleRemove}
              className="block w-full text-left px-4 py-2 hover:bg-gray-100 dark:hover:bg-zinc-700 text-gray-900 dark:text-zinc-100"
            >
              Remove
            </button>
          </div>
        </div>
      )}
    </div>
  );
});

export default MessageItem;