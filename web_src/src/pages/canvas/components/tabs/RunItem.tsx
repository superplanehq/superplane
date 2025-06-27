import React, { JSX, useEffect } from 'react';
import { formatRelativeTime } from '../../utils/stageEventUtils';

interface RunItemProps {
  status: string;
  title: string;
  inputs: Record<string, string>;
  outputs: Record<string, string>;
  timestamp: string;
}

export const RunItem: React.FC<RunItemProps> = React.memo(({ 
  status, 
  title, 
  inputs, 
  outputs,
  timestamp, 
}) => {
  const [isExpanded, setIsExpanded] = React.useState<boolean>(false);
  const [spinChar, setSpinChar] = React.useState<string>('|');

  // Animation for running status
  useEffect(() => {
    if (status.toLowerCase() === 'state_started' || status.toLowerCase() === 'running') {
      const chars = ['|', '/', '—', '\\'];
      let charIndex = 0;
      
      const interval = setInterval(() => {
        charIndex = (charIndex + 1) % chars.length;
        setSpinChar(chars[charIndex]);
      }, 300);

      return () => clearInterval(interval);
    }
  }, [status]);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const renderStatusIcon = (): JSX.Element | null => {
    const statusKey = status.toLowerCase();
    switch (statusKey) {
      case 'state_succeeded':
      case 'passed':
        return (
          <div className="w-5 h-5 rounded-full bg-green-600 mr-2 flex items-center justify-center">
            <span className="material-symbols-outlined text-white text-sm">check_circle</span>
          </div>
        );
      case 'state_failed':
      case 'failed':
        return (
          <div className="w-5 h-5 rounded-full bg-red-600 mr-2 flex items-center justify-center">
            <span className="material-symbols-outlined text-white text-sm">cancel</span>
          </div>
        );
      case 'state_pending':
      case 'queued':
        return (
          <div className="w-5 h-5 rounded-full bg-orange-600 mr-2 flex items-center justify-center">
            <span className="material-symbols-outlined text-white text-sm">queue</span>
          </div>
        );
      case 'state_started':
      case 'running':
        return (
          <div className="w-5 h-5 rounded-full bg-blue-600 mr-2 flex items-center justify-center">
            <span className="text-white text-xs font-mono">{spinChar}</span>
          </div>
        );
      default:
        return (
          <div className="w-5 h-5 rounded-full bg-gray-600 mr-2 flex items-center justify-center">
            <span className="material-symbols-outlined text-white text-sm">help</span>
          </div>
        );
    }
  };

  const getBackgroundClass = (): string => {
    const statusKey = status.toLowerCase();
    switch (statusKey) {
      case 'state_succeeded':
      case 'passed':
        return 'bg-green-50 border-l-4 border-green-500';
      case 'state_failed':
      case 'failed':
        return 'bg-red-50 border-l-4 border-red-500';
      case 'state_started':
      case 'running':
        return 'bg-blue-50 border-l-4 border-blue-500';
      case 'state_pending':
      case 'queued':
        return 'bg-orange-50 border-l-4 border-orange-500';
      default:
        return 'bg-gray-50 border-l-4 border-gray-500';
    }
  };

  return (
    <div className={`mb-2 bg-white rounded-lg border border-gray-200 overflow-hidden`}>
     <div className={`flex w-full items-start p-3 ${getBackgroundClass()}`}>
      <button 
          className="p-1 hover:bg-gray-100 rounded mr-2 text-gray-600"
          onClick={toggleExpand}
          title={isExpanded ? "Hide details" : "Show details"}
        >
          <span className="material-symbols-outlined text-sm">{isExpanded ? 'expand_more' : 'chevron_right'}</span>
      </button>
      <div className='w-full'>
        <div className="flex justify-between items-center">
          <div className="flex items-center min-w-0 flex-1">
            {renderStatusIcon()}
            <span className="font-semibold text-gray-900 truncate">{title}</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="text-xs text-gray-500">{formatRelativeTime(timestamp)}</div>
            <button className="p-1 hover:bg-gray-100 rounded text-gray-400">
              <span className="material-symbols-outlined text-sm">more_vert</span>
            </button>
          </div>
        </div>
        
        {/* Collapsed view - show key inputs/outputs */}
        {!isExpanded && Object.keys(inputs).length > 0 && (
          <div className="flex flex-wrap gap-1 mt-2">
            {Object.entries(inputs).slice(0, 3).map(([key, value]) => (
              <span key={key} className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full border">
                {key}: {value}
              </span>
            ))}
            {Object.keys(inputs).length > 3 && (
              <span className="text-xs text-gray-500">+{Object.keys(inputs).length - 3} more</span>
            )}
          </div>
        )}
         
        {/* Expanded view */}
        {isExpanded && (
          <div className="mt-4 space-y-4">
            <div className="flex items-center space-x-4 text-sm text-gray-600">
              <div className="flex items-center">
                <span className="material-symbols-outlined text-sm mr-1">schedule</span>
                {new Date(timestamp).toLocaleDateString()}
              </div>
              <div className="flex items-center">
                <span className="material-symbols-outlined text-sm mr-1">timer</span>
                Duration info
              </div>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Inputs Section */}
              {Object.keys(inputs).length > 0 && (
                <div>
                  <div className="flex items-center mb-2">
                    <span className="material-symbols-outlined text-sm mr-2 text-gray-600">input</span>
                    <span className="text-sm font-medium text-gray-900 uppercase tracking-wide">Inputs</span>
                  </div>
                  <div className="space-y-1">
                    {Object.entries(inputs).map(([key, value]) => (
                      <div key={key} className="flex justify-between py-1 px-2 bg-gray-50 rounded text-sm font-mono">
                        <span className="text-gray-600">{key}</span>
                        <span className="text-gray-900">{value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
        
              {/* Outputs Section */}
              {Object.keys(outputs).length > 0 && (
                <div className="border-l border-gray-200 pl-4">
                  <div className="flex items-center mb-2">
                    <span className="material-symbols-outlined text-sm mr-2 text-gray-600">output</span>
                    <span className="text-sm font-medium text-gray-900 uppercase tracking-wide">Outputs</span>
                  </div>
                  <div className="space-y-1">
                    {Object.entries(outputs).map(([key, value]) => (
                      <div key={key} className="flex justify-between py-1 px-2 bg-gray-50 rounded text-sm font-mono">
                        <span className="text-gray-600">{key}</span>
                        <span className="text-gray-900">{value}</span>
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