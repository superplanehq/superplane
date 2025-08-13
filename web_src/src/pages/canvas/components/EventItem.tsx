import React, { useState } from 'react';
import { formatRelativeTime, formatFullTimestamp } from '../utils/stageEventUtils';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { PayloadModal } from './PayloadModal';
import { Button } from '@/components/Button/button';
import Tippy from '@tippyjs/react';

interface EventItemProps {
  eventId: string;
  timestamp: string;
}

export const EventItem: React.FC<EventItemProps> = React.memo(({
  eventId,
  timestamp,
}) => {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [showPayloadModal, setShowPayloadModal] = useState<boolean>(false);
  const [showHeadersModal, setShowHeadersModal] = useState<boolean>(false);

  const toggleExpand = (): void => {
    setIsExpanded(!isExpanded);
  };

  const mockHeaders = {
    'Content-Type': 'application/json',
    'User-Agent': 'GitHub-Hookshot/123abc',
    'X-GitHub-Event': 'push',
    'X-GitHub-Delivery': 'abc123-def456-ghi789'
  };

  const mockPayload = {
    ref: 'refs/heads/main',
    before: 'abc123def456',
    after: 'def456ghi789',
    repository: {
      name: 'my-repo',
      full_name: 'user/my-repo',
      private: false
    },
    pusher: {
      name: 'john-doe',
      email: 'john@example.com'
    },
    commits: [
      {
        id: 'def456ghi789',
        message: 'Add new feature',
        author: {
          name: 'John Doe',
          email: 'john@example.com'
        }
      }
    ]
  };

  return (
    <>
      <div className="mb-2 bg-white dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 overflow-hidden rounded-lg">
        <div className="flex w-full items-start p-3">
          <div className='w-full cursor-pointer' onClick={toggleExpand}>
            <div className="flex justify-between items-center">
              <div className="flex items-center min-w-0 flex-1">
                <div className="w-5 h-5 rounded-full mr-2 flex items-center justify-center">
                  <MaterialSymbol name="bolt" size='lg' className="text-blue-600 dark:text-blue-400" />
                </div>
                <span className="font-semibold text-sm text-gray-900 dark:text-zinc-100 truncate">{eventId}</span>
              </div>
              <div className="flex items-center gap-2">
                {!isExpanded && (
                  <Tippy content={formatFullTimestamp(timestamp)} placement="top">
                    <div className="text-xs text-gray-500 dark:text-zinc-400 cursor-help">{formatRelativeTime(timestamp)}</div>
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
                <div className="grid grid-cols-1 gap-4 text-xs p-4 rounded-md bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
                  <div>
                    <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Event ID</div>
                    <div className="font-medium text-gray-900 dark:text-zinc-300 font-mono">{eventId}</div>
                  </div>
                  <div>
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
        content={mockHeaders}
      />

      <PayloadModal
        isOpen={showPayloadModal}
        onClose={() => setShowPayloadModal(false)}
        title="Event Payload"
        content={mockPayload}
      />
    </>
  );
});