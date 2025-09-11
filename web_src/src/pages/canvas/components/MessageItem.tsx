import React, { useCallback, useMemo, useRef, useState } from 'react';
import { SuperplaneStageEvent, SuperplaneStage, SuperplaneEvent } from '@/api-client';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { formatRelativeTime } from '../utils/stageEventUtils';
import { PayloadDisplay } from './PayloadDisplay';

interface MessageItemProps {
  event: SuperplaneStageEvent | SuperplaneEvent;
  sourceEvent?: SuperplaneEvent;
  selectedStage?: SuperplaneStage;
  onApprove?: (eventId: string) => void;
  onCancel?: (eventId: string) => void;
  onRemove?: (eventId: string) => void;
  executionRunning?: boolean;
  plainEventPayload?: { [key: string]: unknown };
  plainEventHeaders?: { [key: string]: unknown };
  discardedOn?: string;
  discardedBy?: string;
  approvedOn?: string;
  approvedBy?: string;
}

const MessageItem = React.memo(({
  event,
  selectedStage,
  onApprove,
  onCancel,
  onRemove,
  plainEventPayload,
  plainEventHeaders,
  sourceEvent,
  discardedOn,
  discardedBy,
  approvedOn,
  approvedBy
}: MessageItemProps) => {
  const [isExpanded, setIsExpanded] = React.useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Type guard to determine if this is a SuperplaneStageEvent
  const isStageEvent = (evt: SuperplaneStageEvent | SuperplaneEvent): evt is SuperplaneStageEvent => {
    return 'approvals' in evt;
  };

  // Type guard to determine if this is a SuperplaneEvent (discarded event)
  const isPlainEvent = (evt: SuperplaneStageEvent | SuperplaneEvent): evt is SuperplaneEvent => {
    return !isStageEvent(evt);
  };

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

  const handleCancel = () => {
    if (onCancel && event.id) {
      onCancel(event.id);
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
    // Only show inputs for stage events, not plain events
    if (isPlainEvent(event) || !selectedStage) {
      return {};
    }

    const map: Record<string, string> = {};
    const stageEvent = event as SuperplaneStageEvent;
    const eventInputs = stageEvent.inputs?.map(input => [input.name, input.value]).reduce((acc, [key, value]) => {
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
    selectedStage?.spec?.conditions
      ?.find(condition => condition.type === "CONDITION_TYPE_APPROVAL")
      ?.approval?.count || 0,
    [selectedStage]);

  const timeWindowCondition = useMemo(() =>
    selectedStage?.spec?.conditions
      ?.find(condition => condition.type === "CONDITION_TYPE_TIME_WINDOW"),
    [selectedStage]);


  const formatTimeWindow = useCallback(() => {
    if (timeWindowCondition?.timeWindow?.start && timeWindowCondition?.timeWindow?.end) {
      // Simple formatting - you can enhance this based on your needs
      return `Run between ${timeWindowCondition.timeWindow.start} and ${timeWindowCondition.timeWindow.end} on ${timeWindowCondition.timeWindow.weekDays?.join(', ')}`;
    }
    return 'Run at scheduled time';
  }, [timeWindowCondition]);

  const getRelativeTime = useCallback(() => {
    let timestamp: string | undefined;
    if (isStageEvent(event)) {
      timestamp = event.createdAt;
    } else {
      timestamp = event.receivedAt;
    }
    if (!timestamp) return 'now';
    return formatRelativeTime(timestamp, true);
  }, [event]);

  return (
    <div className="queueItem">
      <div className="p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-pointer" onClick={toggleExpand}>
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 truncate">
            {isStageEvent(event) && event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_APPROVAL' ? (
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-amber-400/20 text-amber-700 group-data-hover:bg-amber-400/30 dark:bg-amber-400/10 dark:text-amber-400 dark:group-data-hover:bg-amber-400/15">
                <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-base animate-pulse" aria-hidden="true">how_to_reg</span>
                <span className="uppercase">Approval</span>
              </span>
            ) : isStageEvent(event) && event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_TIME_WINDOW' ? (
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-base animate-pulse" aria-hidden="true">schedule</span>
                <span className="uppercase">Scheduled</span>
              </span>
            ) : isStageEvent(event) && event.state === 'STATE_DISCARDED' ? (
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-base" aria-hidden="true">block</span>
                <span className="uppercase">Discarded</span>
              </span>
            ) : isPlainEvent(event) && event.state === 'STATE_DISCARDED' ? (
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-base" aria-hidden="true">block</span>
                <span className="uppercase">Discarded</span>
              </span>
            ) : (
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                <span className="material-symbols-outlined select-none inline-flex items-center justify-center !text-base animate-pulse" aria-hidden="true">pending</span>
                <span className="uppercase">Pending</span>
              </span>
            )}
            <span className="font-medium truncate text-sm dark:text-white">
              {isStageEvent(event) ? (event.name || event.id || 'Unknown') : (event.id || 'Discarded Event')}
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
          <div className="text-left mt-3 space-y-3">
            {/* Queue Section - show for cancelled stage events */}
            {isStageEvent(event) && event.state === 'STATE_DISCARDED' && (event.createdAt || approvedOn || approvedBy || discardedOn || discardedBy) && (
              <div className="space-y-3">
                <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                  Queue
                </div>

                <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-4 text-xs">
                  <div className="space-y-1">
                    {event.createdAt && (
                      <div className="flex items-center gap-1">
                        <MaterialSymbol name="schedule" size="md" className="text-gray-600 dark:text-zinc-400" />
                        <span className="text-xs text-gray-500 dark:text-zinc-400">
                          Added to queue on {new Date(event.createdAt).toLocaleDateString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            year: 'numeric'
                          })} {new Date(event.createdAt).toLocaleTimeString('en-US', {
                            hour: '2-digit',
                            minute: '2-digit',
                            second: '2-digit',
                            hour12: false
                          })}
                        </span>
                      </div>
                    )}
                    {approvedOn && (
                      <div className="flex items-center gap-1">
                        <MaterialSymbol name="check_circle" size="md" className="text-gray-600 dark:text-zinc-400" />
                        <span className="text-xs text-gray-500 dark:text-zinc-400">
                          Approved on {new Date(approvedOn).toLocaleDateString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            year: 'numeric'
                          })} {new Date(approvedOn).toLocaleTimeString('en-US', {
                            hour: '2-digit',
                            minute: '2-digit',
                            second: '2-digit',
                            hour12: false
                          })}
                        </span>
                      </div>
                    )}
                    {approvedBy && (
                      <div className="flex items-center gap-1">
                        <MaterialSymbol name="person" size="md" className="text-gray-600 dark:text-zinc-400" />
                        <span className="text-xs text-gray-500 dark:text-zinc-400 truncate">
                          Approved by <span className="text-blue-600 dark:text-blue-400 truncate">{approvedBy}</span>
                        </span>
                      </div>
                    )}
                    {discardedOn && (
                      <div className="flex items-center gap-1">
                        <MaterialSymbol name="cancel" size="md" className="text-gray-600 dark:text-zinc-400" />
                        <span className="text-xs text-gray-500 dark:text-zinc-400">
                          Discarded on {new Date(discardedOn).toLocaleDateString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            year: 'numeric'
                          })} {new Date(discardedOn).toLocaleTimeString('en-US', {
                            hour: '2-digit',
                            minute: '2-digit',
                            second: '2-digit',
                            hour12: false
                          })}
                        </span>
                      </div>
                    )}
                    {discardedBy && (
                      <div className="flex items-center gap-1">
                        <MaterialSymbol name="person" size="md" className="text-gray-600 dark:text-zinc-400" />
                        <span className="text-xs text-gray-500 dark:text-zinc-400 truncate">
                          Cancelled by <span className="text-blue-600 dark:text-blue-400 truncate">{discardedBy}</span>
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            <div className="mt-3 space-y-3">
              {/* Show payload/headers for plain events */}
              {isPlainEvent(event) && (plainEventPayload || plainEventHeaders) && (
                <div className="space-y-3">
                  <PayloadDisplay
                    showDetailsTab={true}
                    eventId={event.id}
                    timestamp={event.receivedAt}
                    eventType={event.type}
                    sourceName={event.sourceName}
                    headers={plainEventHeaders}
                    payload={plainEventPayload}
                    inputs={inputsRecord}
                    rounded={false}
                  />
                </div>
              )}

              {/* Show payload/headers for stage events */}
              {isStageEvent(event) && sourceEvent && (plainEventPayload || plainEventHeaders) && (
                <div className="space-y-3">
                  {(discardedBy || discardedOn) && <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                    Event
                  </div>}
                  <PayloadDisplay
                    showDetailsTab={true}
                    eventId={sourceEvent.id}
                    timestamp={sourceEvent.receivedAt}
                    eventType={sourceEvent.type}
                    sourceName={sourceEvent.sourceName}
                    headers={plainEventHeaders}
                    payload={plainEventPayload}
                    inputs={inputsRecord}
                    rounded={false}
                  />
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Approval Footer - only for stage events */}
      {isStageEvent(event) && event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_APPROVAL' && (
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
              {/* Cancel Button */}
              {onCancel && (
                <button
                  className="relative isolate inline-flex items-baseline justify-center gap-x-2 rounded-lg border text-base/6 font-semibold px-[calc(--spacing(3.5)-1px)] py-[calc(--spacing(2.5)-1px)] sm:px-[calc(--spacing(3)-1px)] sm:py-[calc(--spacing(1.5)-1px)] sm:text-sm/6 focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-blue-500 data-disabled:opacity-50 *:data-[slot=icon]:-mx-0.5 *:data-[slot=icon]:my-0.5 *:data-[slot=icon]:size-5 *:data-[slot=icon]:shrink-0 *:data-[slot=icon]:self-center *:data-[slot=icon]:text-(--btn-icon) sm:*:data-[slot=icon]:my-1 sm:*:data-[slot=icon]:size-4 forced-colors:[--btn-icon:ButtonText] forced-colors:data-hover:[--btn-icon:ButtonText] border-transparent text-zinc-950 data-active:bg-zinc-950/5 data-hover:bg-zinc-950/5 dark:text-white dark:data-active:bg-white/10 dark:data-hover:bg-white/10 [--btn-icon:var(--color-zinc-500)] data-active:[--btn-icon:var(--color-zinc-700)] data-hover:[--btn-icon:var(--color-zinc-700)] dark:[--btn-icon:var(--color-zinc-500)] dark:data-active:[--btn-icon:var(--color-zinc-400)] dark:data-hover:[--btn-icon:var(--color-zinc-400)] cursor-default"
                  type="button"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleCancel();
                  }}
                >
                  <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
                  <MaterialSymbol name="close" size="sm" className="text-black-700 dark:text-black-400" />
                </button>
              )}

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

      {/* Time Window Footer - only for stage events */}
      {isStageEvent(event) && event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_TIME_WINDOW' && (
        <div className="px-3 py-2 border border-t-0 bg-orange-50 dark:bg-orange-900/20 border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center justify-between">
            <div className="flex items-center text-left">
              <MaterialSymbol name="schedule" size="md" className="text-orange-700 dark:text-orange-200 mr-2" />
              <span className="text-xs text-gray-700 dark:text-zinc-400">{formatTimeWindow()}</span>
            </div>
            <span onClick={() => onCancel?.(event.id || '')} className="text-xs text-black dark:text-zinc-400 cursor-pointer underline">Cancel</span>
          </div>
        </div>
      )}

      {/* Time Window Footer - only for stage events */}
      {isStageEvent(event) && event.state === 'STATE_PENDING' && (
        <div className="px-3 py-2 border border-t-0 bg-orange-50 dark:bg-orange-900/20 border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-700 dark:text-zinc-400">Waiting for execution</span>
            <span onClick={() => onCancel?.(event.id || '')} className="text-xs text-black dark:text-zinc-400 cursor-pointer underline">Cancel</span>
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