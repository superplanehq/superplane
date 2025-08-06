import React, { JSX } from 'react';
import { formatRelativeTime } from '../../utils/stageEventUtils';
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
}

export const RunItem: React.FC<RunItemProps> = React.memo(({
  state,
  result,
  title,
  timestamp,
  executionDuration,
  inputs,
  outputs,
}) => {
  const [isExpanded, setIsExpanded] = React.useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const renderStatusIcon = (): JSX.Element | null => {
    switch (state) {
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return (
            <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
              <MaterialSymbol name="check_circle" size='lg' className="text-green-600 dark:text-green-400" />
            </div>
          );
        }
        if (result === 'RESULT_FAILED') {
          return (
            <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
              <MaterialSymbol name="cancel" size='lg' className="text-red-600 dark:text-red-400" />
            </div>
          );
        }
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="help" size="lg" className="text-gray-600 dark:text-gray-400" />
          </div>
        );
      case 'STATE_PENDING':
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="hourglass" size="lg" className="text-orange-600 dark:text-orange-400" />
          </div>
        );
      case 'STATE_STARTED':
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center animate-spin">
            <MaterialSymbol name="sync" size="lg" className="text-blue-600 dark:text-blue-400" />
          </div>
        );
      default:
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="help" size="lg" className="text-gray-600 dark:text-gray-400" />
          </div>
        );
    }
  };

  const getBackgroundClass = (): string => {
    switch (state) {
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return 'bg-green-50 dark:bg-green-900/50 border-t-1 border-green-500 dark:border-green-700';
        }
        if (result === 'RESULT_FAILED') {
          return 'bg-red-50 dark:bg-red-900/50 border-t-1 border-red-500 dark:border-red-700';
        }
        return 'bg-gray-50 dark:bg-gray-900/50 border-t-1 border-gray-500 dark:border-gray-700';
      case 'STATE_PENDING':
        return 'bg-yellow-50 dark:bg-yellow-900/50 border-t-1 border-yellow-500 dark:border-yellow-700';
      case 'STATE_STARTED':
        return 'bg-blue-50 dark:bg-blue-900/50 border-t-1 border-blue-500 dark:border-blue-700';
      default:
        return 'bg-gray-50 dark:bg-gray-900/50 border-t-1 border-gray-500 dark:border-gray-700';
    }
  };


  return (
    <div className={`mb-2 bg-white dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 overflow-hidden`}>
      <div className={`flex w-full items-start p-2 ${getBackgroundClass()}`}>

        <div className='w-full cursor-pointer' onClick={toggleExpand}>
          <div className="flex justify-between items-center">
            <div className="flex items-center min-w-0 flex-1">
              {renderStatusIcon()}
              <span className="font-semibold text-sm text-gray-900 dark:text-zinc-100 truncate">{title}</span>
            </div>
            <div className="flex items-center gap-2">
              {
                !isExpanded && (
                  <div className="text-xs text-gray-500 dark:text-zinc-400">{formatRelativeTime(timestamp)}</div>
                )
              }
            </div>
            <button
              className='pt-[3px]'
              title={isExpanded ? "Hide details" : "Show details"}
            >
              <MaterialSymbol name={isExpanded ? 'expand_less' : 'expand_more'} size="lg" className="text-gray-600 dark:text-gray-400" />
            </button>
          </div>

          {/* Expanded content */}
          {isExpanded && (
            <div className="mt-3 space-y-3 text-left">
              <div className="grid grid-cols-2 gap-4 text-xs p-4 rounded-md bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
                <div>
                  <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">State</div>
                  <div className="font-semibold text-blue-600 dark:text-blue-400">{state.split('_').at(-1)}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Result</div>
                  <div className="font-semibold text-blue-600 dark:text-blue-400">{result.split('_').at(-1)}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Duration</div>
                  <div className="font-medium text-gray-900 dark:text-zinc-300 font-mono">{executionDuration || "00h 00m 00s"}</div>
                </div>
                <div>
                  <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Started on</div>
                  <div className="font-medium text-gray-900 dark:text-zinc-300">
                    {new Date(timestamp).toLocaleDateString('en-US', {
                      month: 'short',
                      day: 'numeric',
                      year: 'numeric'
                    }) + ' ' + new Date(timestamp).toLocaleTimeString('en-US', {
                      hour: '2-digit',
                      minute: '2-digit',
                      second: '2-digit',
                      hour12: false
                    })}
                  </div>
                </div>
              </div>

              {Object.keys(inputs).length > 0 && (
                <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
                  <div className="flex items-start gap-3">
                    <div className="flex-1">
                      <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Inputs</div>
                      <div className="space-y-1">
                        {Object.entries(inputs).map(([key, value]) => (
                          <div key={key} className="flex items-center justify-between">
                            <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">{key}</span>
                            <div className="flex items-center gap-2 truncate">
                              <span className="font-mono !text-xs truncate inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                                {value}
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
                <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
                  <div className="flex items-start gap-3">
                    <div className="flex-1">
                      <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Outputs</div>
                      <div className="space-y-1">
                        {Object.entries(outputs).map(([key, value]) => (
                          <div key={key} className="flex items-center justify-between">
                            <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">{key}</span>
                            <div className="flex items-center gap-2 truncate">
                              <span className="font-mono !text-xs truncate inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                                {value}
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});