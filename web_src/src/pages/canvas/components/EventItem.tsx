import React, { useState } from 'react';
import { formatFullTimestamp, formatRelativeTime } from '../utils/stageEventUtils';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { PayloadModal } from './PayloadModal';
import { Button } from '@/components/Button/button';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';

interface EventItemProps {
  eventId: string;
  timestamp: string;
  state?: string;
  eventType?: string;
  sourceName?: string;
  headers?: { [key: string]: unknown };
  payload?: { [key: string]: unknown };
}

export const EventItem: React.FC<EventItemProps> = React.memo(({
  eventId,
  timestamp,
  state,
  eventType,
  sourceName,
  headers,
  payload,
}) => {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [showPayloadModal, setShowPayloadModal] = useState<boolean>(false);
  const [showHeadersModal, setShowHeadersModal] = useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const displayHeaders = headers || {};
  const displayPayload = payload || {};

  const renderStateIcon = () => {
    switch (state) {
      case 'processed':
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="check_circle" size='lg' className="text-green-600 dark:text-green-400" />
          </div>
        );
      case 'pending':
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="hourglass" size="lg" className="text-orange-600 dark:text-orange-400" />
          </div>
        );
      case 'discarded':
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="cancel" size='lg' className="text-red-600 dark:text-red-400" />
          </div>
        );
      default:
        return (
          <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
            <MaterialSymbol name="bolt" size='lg' className="text-blue-600 dark:text-blue-400" />
          </div>
        );
    }
  };

  const getBackgroundClass = () => {
    switch (state) {
      case 'processed':
        return 'bg-green-50 dark:bg-green-900/50 border-t-1 border-green-500 dark:border-green-700';
      case 'pending':
        return 'bg-yellow-50 dark:bg-yellow-900/50 border-t-1 border-yellow-500 dark:border-yellow-700';
      case 'discarded':
        return 'bg-red-50 dark:bg-red-900/50 border-t-1 border-red-500 dark:border-red-700';
      default:
        return 'bg-blue-50 dark:bg-blue-900/50 border-t-1 border-blue-500 dark:border-blue-700';
    }
  };

  return (
    <>
      <div className="mb-2 bg-white dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 overflow-hidden">
        <div className={`flex w-full items-start p-2 ${getBackgroundClass()}`}>
          <div className='w-full cursor-pointer' onClick={toggleExpand}>
            <div className="flex justify-between items-center">
              <div className="flex items-center min-w-0 flex-1">
                {renderStateIcon()}
                <span className="font-semibold text-sm text-gray-900 dark:text-zinc-100 truncate">{eventId}</span>
              </div>
              <div className="flex items-center gap-2">
                {!isExpanded && (
                  <Tippy content={formatFullTimestamp(timestamp)} placement="top">
                    <div className="text-xs text-gray-500 dark:text-zinc-400">{formatRelativeTime(timestamp)}</div>
                  </Tippy>
                )}
              </div>
              <button
                className='pt-[3px]'
                title={isExpanded ? "Hide details" : "Show details"}
              >
                <MaterialSymbol name={isExpanded ? 'expand_less' : 'expand_more'} size="lg" className="text-gray-600 dark:text-gray-400" />
              </button>
            </div>

            {isExpanded && (
              <div className="mt-3 space-y-3 text-left">
                <div className="grid grid-cols-2 gap-4 text-xs p-4 rounded-md bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
                  <div>
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">State</div>
                    <div className="font-semibold text-blue-600 dark:text-blue-400">{state?.toUpperCase() || 'UNKNOWN'}</div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Type</div>
                    <div className="font-semibold text-blue-600 dark:text-blue-400">{eventType || 'UNKNOWN'}</div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Event ID</div>
                    <div className="font-medium text-gray-900 dark:text-zinc-300 font-mono">{eventId}</div>
                  </div>
                  <div>
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Source</div>
                    <div className="font-medium text-gray-900 dark:text-zinc-300">{sourceName || 'Unknown'}</div>
                  </div>
                  <div className="col-span-2">
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Received on</div>
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

                <div className="flex gap-2">
                  <Button
                    color="blue"
                    onClick={(e: React.MouseEvent) => {
                      e.stopPropagation();
                      setShowHeadersModal(true);
                    }}
                    className="flex-1"
                  >
                    View Headers
                  </Button>
                  <Button
                    color="green"
                    onClick={(e: React.MouseEvent) => {
                      e.stopPropagation();
                      setShowPayloadModal(true);
                    }}
                    className="flex-1"
                  >
                    View Payload
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <PayloadModal
        isOpen={showHeadersModal}
        onClose={() => setShowHeadersModal(false)}
        title="Event Headers"
        content={displayHeaders}
      />

      <PayloadModal
        isOpen={showPayloadModal}
        onClose={() => setShowPayloadModal(false)}
        title="Event Payload"
        content={displayPayload}
      />
    </>
  );
});