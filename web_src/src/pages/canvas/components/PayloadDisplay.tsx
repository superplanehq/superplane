import React, { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { twMerge } from 'tailwind-merge';
import { PayloadModal } from './PayloadModal';

interface PayloadDisplayProps {
  headers?: { [key: string]: unknown };
  payload?: { [key: string]: unknown };
  // Details tab props
  eventId?: string;
  timestamp?: string;
  eventType?: string;
  sourceName?: string;
  showDetailsTab?: boolean;
  // Inputs/Outputs props
  inputs?: Record<string, string>;
  outputs?: Record<string, string>;
  rounded?: boolean;
}

type TabType = 'details' | 'headers' | 'payload' | 'inputs' | 'outputs';

export const PayloadDisplay: React.FC<PayloadDisplayProps> = ({
  headers,
  payload,
  eventId,
  timestamp,
  eventType,
  sourceName,
  showDetailsTab = false,
  inputs,
  outputs,
  rounded = true,
}) => {
  const displayHeaders = headers || {};
  const displayPayload = payload || {};
  const displayInputs = inputs || {};
  const displayOutputs = outputs || {};

  const hasHeaders = Object.keys(displayHeaders).length > 0;
  const hasPayload = Object.keys(displayPayload).length > 0;
  const hasInputs = Object.keys(displayInputs).length > 0;
  const hasOutputs = Object.keys(displayOutputs).length > 0;

  const getDefaultTab = (): TabType => {
    if (hasInputs) return 'inputs';
    if (hasOutputs) return 'outputs';
    if (showDetailsTab) return 'details';
    if (hasHeaders) return 'headers';
    if (hasPayload) return 'payload';
    return 'details';
  };

  const [activeTab, setActiveTab] = useState<TabType>(getDefaultTab());
  const [isModalOpen, setIsModalOpen] = useState(false);


  const formatTimestamp = () => {
    if (!timestamp) return 'N/A';
    return new Date(timestamp).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric'
    }) + ', ' + new Date(timestamp).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: true
    });
  };


  const renderTabButton = (tabKey: TabType, label: string) => {
    const isActive = activeTab === tabKey;
    return (
      <button
        type="button"
        className={`relative flex items-center gap-2 font-medium transition-all duration-200 ease-in-out focus:outline-hidden text-xs px-3 py-2 relative ${isActive
          ? 'text-blue-600 dark:text-blue-400 cursor-pointer'
          : 'text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 cursor-pointer'
          }`}
        onClick={(e) => {
          e.stopPropagation();
          setActiveTab(tabKey);
        }}
        data-testid={`tab-${tabKey}`}
      >
        <span className="leading-none whitespace-nowrap">{label}</span>
        <div
          className={`absolute inset-x-0 bottom-0 h-0.5 bg-blue-500 transition-all duration-200 ease-in-out ${isActive ? 'scale-x-100' : 'scale-x-0'
            }`}
        ></div>
      </button>
    );
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 'details':
        return (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-6 text-sm">
              <div>
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">RECEIVED ON</div>
                <div className="text-xs text-gray-900 dark:text-zinc-200">{formatTimestamp()}</div>
              </div>
              <div>
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">SOURCE</div>
                <div className="text-xs text-gray-900 dark:text-zinc-200">{sourceName || 'External Webhook'}</div>
              </div>
              <div>
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">TYPE</div>
                <div className="text-xs text-gray-900 dark:text-zinc-200 font-medium">{eventType || 'webhook'}</div>
              </div>
              <div>
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">EVENT ID</div>
                <div className="font-mono text-xs text-gray-900 dark:text-zinc-200 break-all">{eventId}</div>
              </div>
            </div>
          </div>
        );
      case 'headers':
        return (
          <div>
            <div className="space-y-2">
              <div className="bg-zinc-50 dark:bg-zinc-800 rounded border border-gray-200 dark:border-zinc-700 p-3 max-h-60 overflow-y-auto">
                {Object.keys(displayHeaders).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(displayHeaders).map(([key, value]) => (
                      <div key={key} className="flex justify-between">
                        <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium pr-2 flex-shrink-0">
                          {key}
                        </span>
                        <span className="text-xs font-mono text-gray-900 dark:text-zinc-200 break-all">
                          {String(value)}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-gray-500 dark:text-zinc-400 italic">
                    No headers available
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      case 'payload':
        return (
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-gray-500 dark:text-zinc-400">Event Data</span>
              <div className="flex items-center">
                <a
                  className="!text-xs flex items-center cursor-pointer"
                  href="#"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setIsModalOpen(true);
                  }}
                >
                  <MaterialSymbol name="fullscreen" size="sm" className="mr-1" />
                  Expand
                </a>
              </div>
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 rounded border border-gray-200 dark:border-zinc-700 p-3 max-h-60 overflow-y-auto">
              {Object.keys(displayPayload).length > 0 ? (
                <pre className="text-xs font-mono text-gray-900 dark:text-zinc-200 whitespace-pre-wrap">
                  {JSON.stringify(displayPayload, null, 2)}
                </pre>
              ) : (
                <div className="text-xs text-gray-500 dark:text-zinc-400 italic">
                  No payload available
                </div>
              )}
            </div>
          </div>
        );
      case 'inputs':
        return (
          <div>
            <div className="space-y-2">
              <div className="bg-zinc-50 dark:bg-zinc-800 rounded border border-gray-200 dark:border-zinc-700 p-3 max-h-60 overflow-y-auto">
                {Object.keys(displayInputs).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(displayInputs).map(([key, value]) => (
                      <div key={key} className="flex justify-between">
                        <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium pr-2 flex-shrink-0">
                          {key}
                        </span>
                        <span className="text-xs font-mono text-gray-900 dark:text-zinc-200 break-all">
                          {value || '-'}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-gray-500 dark:text-zinc-400 italic">
                    No inputs available
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      case 'outputs':
        return (
          <div>
            <div className="space-y-2">
              <div className="bg-zinc-50 dark:bg-zinc-800 rounded border border-gray-200 dark:border-zinc-700 p-3 max-h-60 overflow-y-auto">
                {Object.keys(displayOutputs).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(displayOutputs).map(([key, value]) => (
                      <div key={key} className="flex justify-between">
                        <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium pr-2 flex-shrink-0">
                          {key}
                        </span>
                        <span className="text-xs font-mono text-gray-900 dark:text-zinc-200 break-all">
                          {value || '-'}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-gray-500 dark:text-zinc-400 italic">
                    No outputs available
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      default:
        return null;
    }
  };

  // Don't render if no data to display
  if (!hasHeaders && !hasPayload && !hasInputs && !hasOutputs && !showDetailsTab) {
    return null;
  }

  return (
    <>
      <div className="mt-3" onClick={(e) => e.stopPropagation()}>
        <div className={twMerge(`border border-gray-200 dark:border-zinc-700 bg-white dark:bg-zinc-900`, rounded ? 'rounded-lg' : '')}>
          <div className="border-b border-gray-200 dark:border-zinc-700">
            <div className="w-full border-b border-zinc-200 dark:border-zinc-700">
              <nav className="flex gap-0">
                {hasInputs && renderTabButton('inputs', 'Inputs')}
                {hasOutputs && renderTabButton('outputs', 'Outputs')}
                {showDetailsTab && renderTabButton('details', 'Details')}
                {hasHeaders && renderTabButton('headers', 'Headers')}
                {hasPayload && renderTabButton('payload', 'Payload')}
              </nav>
            </div>
          </div>
          <div className="px-4 py-3">{renderTabContent()}</div>
        </div>
      </div>
      
      <PayloadModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Event Payload"
        content={displayPayload}
      />
    </>
  );
};