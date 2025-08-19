import React, { JSX } from 'react';
import { ExecutionResult, SuperplaneExecutionState } from '@/api-client';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface RunItemProps {
  state: SuperplaneExecutionState;
  result: ExecutionResult;
  title: string;
  inputs: Record<string, string>;
  outputs: Record<string, string>;
  timestamp: string;
  executionDuration?: string;
  eventId?: string;
  queuedOn?: string;
  approvedOn?: string;
  approvedBy?: string;
}

export const RunItem: React.FC<RunItemProps> = React.memo(({
  state,
  result,
  title,
  timestamp,
  executionDuration,
  inputs,
  outputs,
  eventId,
  queuedOn,
  approvedOn,
  approvedBy,
}) => {
  const [isExpanded, setIsExpanded] = React.useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const renderStatusBadge = (): JSX.Element => {
    switch (state) {
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return (
            <button className="!flex !items-center group relative inline-flex rounded-md focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-green-500 hover:bg-green-500/10" type="button">
              <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-green-500/15 text-green-700 group-hover:bg-green-500/25 dark:text-green-400 dark:group-hover:bg-green-500/25">
                <MaterialSymbol name="check_circle" size="sm" />
                <span className="uppercase">passed</span>
              </span>
            </button>
          );
        }
        if (result === 'RESULT_FAILED') {
          return (
            <button className="!flex !items-center group relative inline-flex rounded-md focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-red-500 hover:bg-red-500/10" type="button">
              <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
              <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-red-500/15 text-red-700 group-hover:bg-red-500/25 dark:text-red-400 dark:group-hover:bg-red-500/25">
                <MaterialSymbol name="cancel" size="sm" />
                <span className="uppercase">failed</span>
              </span>
            </button>
          );
        }
        return (
          <button className="!flex !items-center group relative inline-flex rounded-md focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-gray-500 hover:bg-gray-500/10" type="button">
            <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
            <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-gray-500/15 text-gray-700 group-hover:bg-gray-500/25 dark:text-gray-400 dark:group-hover:bg-gray-500/25">
              <MaterialSymbol name="help" size="sm" />
              <span className="uppercase">finished</span>
            </span>
          </button>
        );
      case 'STATE_PENDING':
        return (
          <button className="!flex !items-center group relative inline-flex rounded-md focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-orange-500 hover:bg-orange-500/10" type="button">
            <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
            <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-orange-500/15 text-orange-700 group-hover:bg-orange-500/25 dark:text-orange-400 dark:group-hover:bg-orange-500/25">
              <MaterialSymbol name="hourglass_empty" size="sm" />
              <span className="uppercase">pending</span>
            </span>
          </button>
        );
      case 'STATE_STARTED':
        return (
          <button className="!flex !items-center group relative inline-flex rounded-md focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-blue-500 hover:bg-blue-500/10" type="button">
            <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
            <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-blue-500/15 text-blue-700 group-hover:bg-blue-500/25 dark:text-blue-400 dark:group-hover:bg-blue-500/25">
              <MaterialSymbol name="sync" size="sm" className="animate-spin" />
              <span className="uppercase">running</span>
            </span>
          </button>
        );
      default:
        return (
          <button className="!flex !items-center group relative inline-flex rounded-md focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-gray-500 hover:bg-gray-500/10" type="button">
            <span className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden" aria-hidden="true"></span>
            <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-gray-500/15 text-gray-700 group-hover:bg-gray-500/25 dark:text-gray-400 dark:group-hover:bg-gray-500/25">
              <MaterialSymbol name="help" size="sm" />
              <span className="uppercase">unknown</span>
            </span>
          </button>
        );
    }
  };

  const getBorderColor = (): string => {
    switch (state) {
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return 'border-t-green-400 dark:border-t-green-700';
        }
        if (result === 'RESULT_FAILED') {
          return 'border-t-red-400 dark:border-t-red-700';
        }
        return 'border-t-gray-400 dark:border-t-gray-700';
      case 'STATE_PENDING':
        return 'border-t-orange-400 dark:border-t-orange-700';
      case 'STATE_STARTED':
        return 'border-t-blue-400 dark:border-t-blue-700';
      default:
        return 'border-t-gray-400 dark:border-t-gray-700';
    }
  };


  return (
    <div className={`border-b border-l border-r border-gray-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 border-t  ${getBorderColor()}`}>
      <div className="p-3">
        <div className="flex items-center justify-between cursor-pointer min-w-0" onClick={toggleExpand}>
          <div className="text-xs gap-2 min-w-0 flex-1">
            <div className="flex items-center gap-2 mb-2">
              {renderStatusBadge()}
              {title && (
                <div className="font-medium text-blue-600 dark:text-blue-400 flex items-center gap-1 text-sm min-w-0">
                  <span className="truncate">{title}</span>
                  <MaterialSymbol name="arrow_outward" size="sm" className="flex-shrink-0" />
                </div>
              )}
            </div>
            <div className="flex items-center gap-4 mb-1">
              <div className="flex items-center gap-1 truncate">
                <MaterialSymbol name="calendar_today" size="md" className="text-gray-600 dark:text-zinc-400" />
                <span className="text-xs text-gray-500 dark:text-zinc-400 truncate">
                  Started on {new Date(timestamp).toLocaleDateString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    year: 'numeric'
                  })} {new Date(timestamp).toLocaleTimeString('en-US', {
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit',
                    hour12: false
                  })}
                </span>
              </div>
              {executionDuration && (
                <div className="flex items-center gap-1 truncate">
                  <MaterialSymbol name="timer" size="md" className="text-gray-600 dark:text-zinc-400" />
                  <span className="text-xs text-gray-500 dark:text-zinc-400 truncate">{executionDuration}</span>
                </div>
              )}
            </div>
            <div className="flex items-center gap-1 min-w-0">
              <MaterialSymbol name="bolt" size="md" className="text-gray-600 dark:text-zinc-400 flex-shrink-0" />
              <span className="text-xs text-gray-500 dark:text-zinc-400 min-w-0 flex items-center">
                <div className="text-blue-600 dark:text-blue-400 truncate">{title}</div>
                {eventId && (
                  <>
                    <span className="mx-1 flex-shrink-0"> • Event ID: </span>
                    <div className="text-blue-600 dark:text-blue-400 truncate">{eventId}</div>
                  </>
                )}
              </span>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <MaterialSymbol
              name={isExpanded ? 'expand_less' : 'expand_more'}
              size="xl"
              className="text-gray-600 dark:text-zinc-400"
            />
          </div>
        </div>

        {/* Expanded content */}
        {isExpanded && (
          <div className="mt-3 space-y-3 text-left">
            {Object.keys(inputs).length > 0 && (
              <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-zinc-50 dark:bg-zinc-800">
                <div className="flex items-start gap-3">
                  <div className="flex-1">
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Inputs</div>
                    <div className="space-y-1">
                      {Object.entries(inputs).map(([key, value]) => (
                        <div key={key} className="flex items-center justify-between gap-2 min-w-0">
                          <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium font-mono truncate">{key}</span>
                          <div className="flex items-center gap-2 flex-shrink-0">
                            <span className="font-mono !text-xs truncate inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10 max-w-32">
                              {value || '-'}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {Object.keys(outputs).length > 0 && (
              <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-zinc-50 dark:bg-zinc-800">
                <div className="flex items-start gap-3">
                  <div className="flex-1">
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Outputs</div>
                    <div className="space-y-1">
                      {Object.entries(outputs).map(([key, value]) => (
                        <div key={key} className="flex items-center justify-between gap-2 min-w-0">
                          <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium font-mono truncate">{key}</span>
                          <div className="flex items-center gap-2 flex-shrink-0">
                            <span className="font-mono !text-xs truncate inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10 max-w-32">
                              {value || '-'}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {(queuedOn || approvedOn || approvedBy) && (
              <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-4 text-xs">
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-2">Queue Information</div>
                <div className="space-y-1">
                  {queuedOn && (
                    <div className="flex items-center gap-1">
                      <MaterialSymbol name="schedule" size="md" className="text-gray-600 dark:text-zinc-400" />
                      <span className="text-xs text-gray-500 dark:text-zinc-400">
                        Added to queue on {new Date(queuedOn).toLocaleDateString('en-US', {
                          month: 'short',
                          day: 'numeric',
                          year: 'numeric'
                        })} {new Date(queuedOn).toLocaleTimeString('en-US', {
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
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
});