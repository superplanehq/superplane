import React, { useCallback, useMemo, useRef, useState } from 'react';
import { SuperplaneStageEvent, SuperplaneStage } from '@/api-client';

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

  return (
    <div className="queueItem">
      <div className="p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-pointer" onClick={toggleExpand}>
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 truncate">
            <span className="font-medium truncate text-sm dark:text-white">
              {event.id || 'Unknown'}
            </span>
          </div>
          <div className="flex items-center gap-3">
            <span className="material-symbols-outlined select-none !text-xl text-gray-600 dark:text-zinc-400">
              {isExpanded ? 'expand_less' : 'expand_more'}
            </span>
          </div>
        </div>

        {isExpanded && (
          <div className="mt-3 space-y-3 text-left">
            <div className="mt-3 space-y-3">
              {Object.keys(inputsRecord).length > 0 ? (
                <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
                  <div className="flex items-start gap-3">
                    <div className="flex-1">
                      <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Inputs</div>
                      <div className="space-y-1">
                        {Object.entries(inputsRecord).map(([key, value]) => (
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
              ) : (
                <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
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

      {event.state === 'STATE_WAITING' && event.stateReason === 'STATE_REASON_APPROVAL' && (
        <div className="px-3 py-2 border border-t-0 bg-orange-50 dark:bg-orange-900/20 border-zinc-200 dark:border-zinc-700">
          <div className="flex justify-between items-center">
            <div className="flex items-center">
              <span className="material-symbols-outlined select-none !text-base text-orange-700 dark:text-orange-200 mr-2">
                how_to_reg
              </span>
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
            {onApprove && (
              <a
                href="#"
                className="text-xs text-gray-700 dark:text-zinc-300 flex items-center"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  handleApprove();
                }}
              >
                <span className="material-symbols-outlined select-none !text-sm text-gray-500 dark:text-zinc-400 mr-1">
                  check
                </span>
                <span className="underline">Approve</span>
              </a>
            )}
          </div>
        </div>
      )}

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