import React, { JSX } from 'react';
import { formatRelativeTime } from '../../utils/stageEventUtils';
import { SuperplaneExecutionResult, SuperplaneExecutionState } from '@/api-client';

interface RunItemProps {
  state: SuperplaneExecutionState;
  result: SuperplaneExecutionResult;
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
              <span className="material-icons text-green-600 text-sm">check_circle</span>
            </div>
          );
        }
        if (result === 'RESULT_FAILED') {
          return (
            <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
              <span className="material-icons text-red-600 text-sm">cancel</span>
            </div>
          );
        }
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <span className="material-icons text-gray-600 text-sm">help</span>
          </div>
        );
      case 'STATE_PENDING':
        return (
          <div className="w-5 h-5 rounded-full bg-orange-500 border border-orange-200 mr-2 flex items-center justify-center">
            <span className="text-white text-xs job-log-pending"></span>
          </div>
        );
      case 'STATE_STARTED':
        return (
          <div className="w-5 h-5 rounded-full bg-blue-500 border border-blue-200 mr-2 flex items-center justify-center">
            <span className="text-white text-xs job-log-working"></span>
          </div>
        );
      default:
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <span className="material-icons text-gray-600 text-sm">help</span>
          </div>
        );
    }
  };

  const getBackgroundClass = (): string => {
    switch (state) {
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return 'bg-green-50 border-t-1 border-green-500';
        }
        if (result === 'RESULT_FAILED') {
          return 'bg-red-50 border-t-1 border-red-500';
        }
        return 'bg-gray-50 border-t-1 border-gray-500';
      case 'STATE_PENDING':
        return 'bg-orange-50 border-t-1 border-orange-500';
      case 'STATE_STARTED':
        return 'bg-blue-50 border-t-1 border-blue-500';
      default:
        return 'bg-gray-50 border-t-1 border-gray-500';
    }
  };

  return (
    <div className={`mb-2 bg-white border border-gray-200 overflow-hidden`}>
     <div className={`flex w-full items-start p-2 ${getBackgroundClass()}`}>
      <button 
          className='pt-[3px]'
          onClick={toggleExpand}
          title={isExpanded ? "Hide details" : "Show details"}
        >
          <span className="material-symbols-outlined text-sm">{isExpanded ? 'arrow_drop_down' : 'arrow_right'}</span>
      </button>
      <div className='w-full'>
        <div className="flex justify-between items-center">
          <div className="flex items-center min-w-0 flex-1">
            {renderStatusIcon()}
            <span className="font-semibold text-gray-900 truncate">{title}</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="text-xs text-gray-500">{formatRelativeTime(timestamp)}</div>
            <button className="p-1 hover:bg-gray-100 rounded text-gray-500">
              <span style={{ fontSize: '15px' }} className="material-symbols-outlined">more_vert</span>
            </button>
          </div>
        </div>
        
        {!isExpanded && Object.keys(outputs).length > 0 && (
          <div className="flex flex-wrap gap-1 mt-2 align-center justify-left">
            {Object.entries(outputs).slice(0, 3).map(([key, value]) => (
              <span key={key} className="bg-gray-200 opacity-70 text-gray-900 text-[12px] px-2 rounded border border-gray-300 font-[family-name:var(--jetbrains-mono)]">
                {key}: {value}
              </span>
            ))}
            {Object.keys(outputs).length > 3 && (
              <span className="text-[11px] text-black pt-[2px] ml-2">+{Object.keys(outputs).length - 3} more</span>
            )}
          </div>
        )}
         
        {/* Expanded view */}
        {isExpanded && (
          <div className="mt-4 space-y-4">
            <div className="flex items-center space-x-4 text-sm text-gray-600">
              <div className="flex items-center text-xs">
                <span style={{ fontSize: '15px' }} className="material-symbols-outlined text-sm mr-1">nest_clock_farsight_analog</span>
                {new Date(timestamp).toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
              </div>
              <div className="flex items-center text-xs">
                <span style={{ fontSize: '15px' }} className="material-symbols-outlined text-sm mr-1">hourglass_bottom</span>
                {executionDuration || "-"}
              </div>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-3">
              {/* Inputs Section */}
              {Object.keys(inputs).length > 0 && (
                <div>
                  <div className="flex items-center mb-2">
                    <span style={{ fontSize: '1em' }} className="material-symbols-outlined text-sm mr-2 text-gray-600">input</span>
                    <span className="text-sm font-medium text-gray-900 uppercase tracking-wide">Inputs</span>
                  </div>
                  <div className="max-w-[200px]">
                    {Object.entries(inputs).map(([key, value]) => (
                      <div key={`input-${key}`} className="flex justify-between px-2 pl-6 rounded text-xs font-mono" >
                        <span className="text-gray-600">{key}</span>
                        <span className="text-gray-900 truncate max-w-[100px]">{value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
        
              {/* Outputs Section */}
              {Object.keys(outputs).length > 0 && (
                <div className="border-l border-gray-200 pl-4">
                  <div className="flex items-center mb-2">
                    <span style={{ fontSize: '1em' }} className="material-symbols-outlined text-sm mr-2 text-gray-600">output</span>
                    <span className="text-sm font-medium text-gray-900 uppercase tracking-wide">Outputs</span>
                  </div>
                  <div className="">
                    {Object.entries(outputs).map(([key, value]) => (
                      <div key={`output-${key}`} className="flex justify-between px-2 pl-6 rounded text-xs font-mono">
                        <span className="text-gray-600">{key}</span>
                        <span className="text-gray-900 truncate max-w-[100px]">{value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
     </div>
    </div>
  );
});