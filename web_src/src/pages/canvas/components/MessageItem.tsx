import React, { useCallback, useMemo, useRef, useState } from 'react';
import { SuperplaneStageEvent, SuperplaneStage } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';

interface MessageItemProps {
  event: SuperplaneStageEvent;
  selectedStage: SuperplaneStage;
  onApprove?: (eventId: string) => void;
  onRemove?: (eventId: string) => void;
  approved?: boolean;
  executionRunning?: boolean;
}

const MessageItem = React.memo(({
  event,
  selectedStage,
  onApprove,
  onRemove,
  approved = false,
}: MessageItemProps) => {
  const [isExpanded, setIsExpanded] = React.useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  const handleDropdownClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsDropdownOpen(!isDropdownOpen);
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

  return (
    <div className="mb-2 bg-white border border-gray-200 overflow-hidden">
      <div className={`flex w-full items-start p-2 bg-white`}>
        <button
          className='pb-[3px]'
          onClick={toggleExpand}
          title={isExpanded ? "Hide details" : "Show details"}
        >
          <span className="material-symbols-outlined text-sm">{isExpanded ? 'arrow_drop_down' : 'arrow_right'}</span>
        </button>
        <div className='w-full'>
          <div className="flex justify-between items-center mb-2">
            <div className="flex items-center min-w-0 flex-1">
              <span
                className="material-icons text-orange-500 mr-2"
                style={{ fontSize: '25px' }}
              >
                input
              </span>
              <span className="font-semibold text-gray-900 truncate">
                Event #{event.id?.substring(0, 8) || 'Unknown'}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="text-xs text-gray-500">
                {event.createdAt ? formatRelativeTime(event.createdAt) : 'Unknown time'}
              </div>
            </div>
          </div>

          {!isExpanded && Object.keys(inputsRecord).length > 0 ? (
            <div className="flex flex-wrap gap-1 mt-2 align-center justify-left mb-1">
              {Object.entries(inputsRecord).slice(0, 3).map(([key, value]) => (
                <span key={key} className="bg-gray-200 opacity-70 text-gray-900 text-[12px] px-2 rounded border border-gray-300 font-[family-name:var(--jetbrains-mono)]">
                  {key}: {value}
                </span>
              ))}
              {Object.keys(inputsRecord).length > 3 && (
                <span className="text-[11px] text-black pt-[2px] ml-2">+{Object.keys(inputsRecord).length - 3} more</span>
              )}
            </div>
          ) : <div className="text-[12px] text-black ml-2 mb-1 text-left w-full">No associated inputs</div>}

          {isExpanded && (
            <div className="mt-4 space-y-4">
              <div className="flex items-center space-x-4 text-sm text-gray-600">
                <div className="flex items-center text-xs">
                  <span style={{ fontSize: '15px' }} className="material-symbols-outlined text-sm mr-1">nest_clock_farsight_analog</span>
                  {event.createdAt ? new Date(event.createdAt).toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : 'Unknown date'}
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3">
                {Object.keys(inputsRecord).length > 0 && (
                  <div>
                    <div className="flex items-center mb-2">
                      <span style={{ fontSize: '1em' }} className="material-symbols-outlined text-sm mr-2 text-gray-600">input</span>
                      <span className="text-sm font-medium text-gray-900 uppercase tracking-wide">Inputs</span>
                    </div>
                    <div className="max-w-[200px]">
                      {Object.entries(inputsRecord).map(([key, value]) => (
                        <div key={`input-${key}`} className="flex justify-between px-2 pl-6 rounded text-xs font-mono" >
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
          {event.state === 'STATE_WAITING' && (
            <div className="flex items-center justify-between pt-2 border-t border-gray-200">
              <div className="flex items-center text-xs">
                <span className="material-symbols-outlined mr-1 text-gray-600" style={{ fontSize: '14px' }}>
                  schedule
                </span>
                Waiting for approval
              </div>

              {event.approvals && event.approvals.length > 0 && (
                <div className="flex items-center text-xs">
                  <span className="material-symbols-outlined text-gray-600" style={{ fontSize: '14px' }}>
                    check_circle
                  </span>
                  <div className="ml-1">
                    approved by <a href="#" className="text-black underline">
                      <span className="underline">
                        {event.approvals.length} person{event.approvals.length !== 1 ? 's' : ''}
                      </span>
                    </a>
                    {event.approvals.length < 3 && ', waiting for more'}
                  </div>
                </div>
              )}


              <div className="flex items-center">
                {onApprove && (
                  <button
                    onClick={handleApprove}
                    disabled={approved}
                    className={"px-3 py-[7px] shadow-sm border rounded-[7px] flex items-center justify-center " + (approved ? "text-green-800 border border-green-800 bg-green-100" : "text-gray-800 border border-gray-200")}
                  >
                    <span style={{ fontSize: '15px' }} className="material-symbols-outlined">check</span>
                  </button>
                )}

                <button
                  className="p-1 hover:bg-gray-100 rounded text-gray-500"
                  onClick={handleDropdownClick}
                >
                  <span style={{ fontSize: '15px' }} className="material-symbols-outlined">more_vert</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>


      {isDropdownOpen && (
        <div
          className="absolute right-0 bg-white shadow-lg rounded-lg w-32 z-10"
          style={{ marginTop: '4px' }}
          ref={dropdownRef}
        >
          <div className="py-1">
            <button
              onClick={handleRemove}
              className="block w-full text-left px-4 py-2 hover:bg-gray-100"
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