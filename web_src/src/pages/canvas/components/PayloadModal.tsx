import { useEffect } from 'react';
import { createPortal } from 'react-dom';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Button } from '@/components/Button/button';

interface PayloadModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  content: Record<string, unknown>;
}

export function PayloadModal({
  isOpen,
  onClose,
  title,
  content
}: PayloadModalProps) {

  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = 'unset';
    }

    return () => {
      document.body.style.overflow = 'unset';
    };
  }, [isOpen]);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const formattedContent = JSON.stringify(content, null, 2);

  const modal = (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-gray-50/55 dark:bg-zinc-900/55">
      <div className="bg-white dark:bg-zinc-800 rounded-lg shadow-xl w-[95vw] h-[95vh] max-w-7xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-zinc-700">
          <div className="flex items-center gap-2">
            <MaterialSymbol name="data_object" size="md" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-zinc-100">
              {title}
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors text-gray-600 dark:text-zinc-400"
            title="Close"
          >
            <MaterialSymbol name="close" size="md" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 border-b border-gray-200 dark:border-zinc-700 overflow-hidden">
          <div className="h-full p-4">
            <div className="bg-gray-50 dark:bg-zinc-900 rounded-lg border border-gray-200 dark:border-zinc-700 h-full overflow-hidden">
              <pre className="p-4 text-sm font-mono text-gray-900 dark:text-zinc-100 overflow-auto h-full whitespace-pre-wrap break-words">
                {formattedContent}
              </pre>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-zinc-800">
          <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-zinc-400">
            <MaterialSymbol name="info" size="sm" />
            <span className='dark:text-zinc-100'>
              {title === 'Event Headers' ? 'HTTP headers received with this event' : 'JSON payload data received with this event'}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              onClick={() => {
                navigator.clipboard.writeText(formattedContent);
              }}
              outline
              className="text-sm"
            >
              <MaterialSymbol name="content_copy" size="sm" data-slot="icon" />
              Copy to Clipboard
            </Button>
            <Button
              onClick={onClose}
              color="blue"
              className="text-sm"
            >
              <MaterialSymbol name="check" size="sm" data-slot="icon" />
              Close
            </Button>
          </div>
        </div>
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}