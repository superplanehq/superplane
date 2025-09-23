import { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import Editor from '@monaco-editor/react';
import { Input } from '@/components/Input/input';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Button } from '@/components/Button/button';
import { SuperplaneEvent } from '@/api-client';

interface EmitEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  sourceName: string;
  loadLastEvent: () => Promise<SuperplaneEvent | null>;
  onCancel?: () => void;
  onSubmit: (eventType: string, eventData: any) => Promise<void>;
}


export function EmitEventModal({ isOpen, onClose, sourceName, loadLastEvent, onCancel, onSubmit }: EmitEventModalProps) {
  const [eventType, setEventType] = useState('');
  const [rawEventData, setRawEventData] = useState('{}');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [isLoadingLastEvent, setIsLoadingLastEvent] = useState(false);

  // Detect dark mode
  useEffect(() => {
    const checkDarkMode = () => {
      setIsDarkMode(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
    };

    checkDarkMode();

    const observer = new MutationObserver(checkDarkMode);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    });

    return () => observer.disconnect();
  }, []);

  // Load last event when modal opens
  useEffect(() => {
    if (isOpen) {
      setIsLoadingLastEvent(true);
      loadLastEvent()
        .then((event) => {
          if (event) {
            setEventType(event.type || '');
            if (event.raw) {
              setRawEventData(JSON.stringify(event.raw, null, 2));
            }
          }
        })
        .catch((error) => {
          console.error('Failed to load last event:', error);
        })
        .finally(() => {
          setIsLoadingLastEvent(false);
        });
    }
  }, [isOpen]);

  const validateAndParseJson = (jsonString: string): { isValid: boolean; parsed?: any } => {
    try {
      const parsed = JSON.parse(jsonString);
      setJsonError(null);
      return { isValid: true, parsed };
    } catch (error) {
      setJsonError('Invalid JSON format');
      return { isValid: false };
    }
  };

  const handleRawDataChange = (value: string | undefined) => {
    const newValue = value || '';
    setRawEventData(newValue);
    if (newValue.trim()) {
      validateAndParseJson(newValue);
    } else {
      setJsonError(null);
    }
  };

  const handleSubmit = async () => {
    if (!eventType.trim()) {
      setError('Event type is required');
      return;
    }

    const { isValid, parsed } = validateAndParseJson(rawEventData);
    if (!isValid) {
      setError('Please fix the JSON format');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await onSubmit(eventType, parsed);

      // Reset form and close modal
      setEventType('');
      setRawEventData('{}');
      setError(null);
      setJsonError(null);
      onClose();
    } catch (error: any) {
      setError(error?.message || 'Failed to emit event');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    if (!isSubmitting) {
      setEventType('');
      setRawEventData('{}');
      setError(null);
      setJsonError(null);
      if (onCancel) {
        onCancel();
      }
      onClose();
    }
  };

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-gray-50/55 dark:bg-zinc-900/55">
      <div className="bg-white dark:bg-zinc-800 rounded-lg shadow-xl w-[85vw] h-[80vh] max-w-5xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-zinc-700">
          <div className="flex items-center gap-3">
            <MaterialSymbol name="send" size="lg" className="text-gray-700 dark:text-gray-300" />
            <div>
              <div className="text-xl font-semibold text-gray-900 dark:text-zinc-100">
                Emit Event
              </div>
              <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Manually emit an event for "{sourceName}"
              </div>
            </div>
          </div>
          <button
            onClick={handleClose}
            disabled={isSubmitting}
            className="p-2 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors text-gray-600 dark:text-zinc-400"
            title="Close"
          >
            <MaterialSymbol name="close" size="md" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 p-6 overflow-hidden flex flex-col">
          {isLoadingLastEvent ? (
            <div className="flex-1 flex items-center justify-center">
              <div className="flex items-center gap-3 text-gray-600 dark:text-gray-400">
                <MaterialSymbol name="hourglass_empty" className="animate-spin" size="lg" />
                <span className="text-lg">Loading last event...</span>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col space-y-4">
              <div>
                <label htmlFor="eventType" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Event Type
                </label>
                <Input
                  id="eventType"
                  value={eventType}
                  onChange={(e) => setEventType(e.target.value)}
                  placeholder="e.g., webhook, push, deployment_complete"
                  disabled={isSubmitting}
                  className="w-full"
                />
              </div>

              <div className="flex-1 flex flex-col">
                <label htmlFor="rawData" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Event Data (JSON)
                </label>
                <div className="flex-1 border border-gray-300 dark:border-gray-600 rounded-md overflow-hidden">
                  <Editor
                    height="100%"
                    defaultLanguage="json"
                    value={rawEventData}
                    onChange={handleRawDataChange}
                    theme={isDarkMode ? 'vs-dark' : 'vs'}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 14,
                      lineNumbers: 'on',
                      rulers: [],
                      wordWrap: 'on',
                      folding: true,
                      bracketPairColorization: {
                        enabled: true
                      },
                      autoIndent: 'advanced',
                      formatOnPaste: true,
                      formatOnType: true,
                      tabSize: 2,
                      insertSpaces: true,
                      scrollBeyondLastLine: false,
                      renderWhitespace: 'boundary',
                      smoothScrolling: true,
                      cursorBlinking: 'smooth',
                      readOnly: isSubmitting,
                      contextmenu: true,
                      selectOnLineNumbers: true
                    }}
                  />
                </div>
                {jsonError && (
                  <p className="text-red-600 dark:text-red-400 text-sm mt-1">
                    {jsonError}
                  </p>
                )}
              </div>

              {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-3">
                  <div className="flex items-center">
                    <MaterialSymbol name="error" className="text-red-400 mr-2" />
                    <span className="text-red-800 dark:text-red-200 text-sm">{error}</span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 p-6 border-t border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-800">
          <Button onClick={handleClose} disabled={isSubmitting} outline>
            Cancel
          </Button>
          <Button
            color="blue"
            onClick={handleSubmit}
            disabled={isSubmitting || isLoadingLastEvent || !eventType.trim() || jsonError !== null}
          >
            {isSubmitting ? (
              <>
                <MaterialSymbol name="hourglass_empty" className="animate-spin" size="sm" />
                Emitting...
              </>
            ) : (
              'Emit Event'
            )}
          </Button>
        </div>
      </div>
    </div>,
    document.body
  );
}