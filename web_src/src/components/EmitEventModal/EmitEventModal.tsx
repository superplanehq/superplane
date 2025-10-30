/**
 * NOTE: This is the EmitEventModal used by the old canvas system (pages/canvas).
 * This component should be removed once the old canvas system is removed.
 * The new system uses @/ui/EmitEventModal instead.
 */
import { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import Editor from '@monaco-editor/react';
import { Input } from '@/components/Input/input';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Button } from '@/components/Button/button';
import { Select } from '@/components/Select';
import { SuperplaneEvent } from '@/api-client';
import { EVENT_TEMPLATES, type EventTemplate } from '@/constants/eventTemplates';

interface EmitEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  sourceName: string;
  nodeType: 'event_source' | 'stage';
  loadLastEvent: () => Promise<SuperplaneEvent | null>;
  onCancel?: () => void;
  onSubmit: (eventType: string, eventData: unknown) => Promise<void>;
}


export function EmitEventModal({ isOpen, onClose, sourceName, nodeType, loadLastEvent, onCancel, onSubmit }: EmitEventModalProps) {
  const [eventType, setEventType] = useState('');
  const [rawEventData, setRawEventData] = useState('{}');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [isLoadingLastEvent, setIsLoadingLastEvent] = useState(false);
  const [hasLastEvent, setHasLastEvent] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState('');
  const availableTemplates = EVENT_TEMPLATES.filter(template => template.nodeType === nodeType);

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
      setHasLastEvent(false);
      setSelectedTemplate('');
      loadLastEvent()
        .then((event) => {
          if (event) {
            setEventType(event.type || '');
            if (event.raw) {
              setRawEventData(JSON.stringify(event.raw, null, 2));
            }
            setHasLastEvent(true);
          } else {
            setHasLastEvent(false);
          }
        })
        .catch((error) => {
          console.error('Failed to load last event:', error);
          setHasLastEvent(false);
        })
        .finally(() => {
          setIsLoadingLastEvent(false);
        });
    }
  }, [isOpen]);

  const validateAndParseJson = (jsonString: string): { isValid: boolean; parsed?: unknown } => {
    try {
      const parsed = JSON.parse(jsonString);
      setJsonError(null);
      return { isValid: true, parsed };
    } catch {
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

  const handleTemplateSelect = (template: EventTemplate) => {
    setEventType(template.eventType);
    setRawEventData(JSON.stringify(template.getEventData(), null, 2));
    setSelectedTemplate(template.name);
    setJsonError(null);
    setError(null);
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
    } catch (error) {
      setError((error as Error)?.message || 'Failed to emit event');
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
      setHasLastEvent(false);
      setSelectedTemplate('');
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
        {/* Body */}
        <div className="flex-1 px-6 py-6 overflow-hidden flex flex-col">
          {isLoadingLastEvent ? (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center py-8">
                <div className="inline-flex items-center justify-center w-16 h-16 mb-3">
                  <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent"></div>
                </div>
                <p className="text-zinc-600 dark:text-zinc-400 text-sm">Loading last emitted event...</p>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col space-y-4">
              <div className="flex items-center gap-2 mb-4">
                <MaterialSymbol
                  name={hasLastEvent ? "history" : "info"}
                  size="md"
                  className={hasLastEvent ? "text-green-600 dark:text-green-400" : "text-blue-600 dark:text-blue-400"}
                />
                <div className="text-sm text-gray-900 dark:text-zinc-100">
                  {hasLastEvent
                    ? <>{`Last event loaded. Modify it as needed before emitting a new event for ${nodeType === 'event_source' ? 'event source' : 'stage'} `}<span className="font-mono bg-yellow-50 dark:bg-yellow-900/20 px-2 py-1 rounded">{sourceName}</span></>
                    : <>{`No events emitted yet. Choose a template to emit your first event for ${nodeType === 'event_source' ? 'event source' : 'stage'} `}<span className="font-mono bg-yellow-50 dark:bg-yellow-900/20 px-2 py-1 rounded">{sourceName}</span></>
                  }
                </div>
              </div>

              {!hasLastEvent && (
                <div className="flex items-center gap-3">
                  <MaterialSymbol name="library_add" size="sm" className="text-gray-600 dark:text-gray-400" />
                  <label className="text-xs font-medium text-gray-700 dark:text-gray-300 whitespace-nowrap w-18">
                    Template
                  </label>
                  <div className="flex-1">
                    <Select
                      options={availableTemplates.map(template => ({
                        value: template.name,
                        label: template.name,
                        description: template.description,
                        icon: template.icon,
                        image: template.image
                      }))}
                      value={selectedTemplate}
                      onChange={(templateName) => {
                        const template = availableTemplates.find(t => t.name === templateName);
                        if (template) handleTemplateSelect(template);
                      }}
                      placeholder="Select a template..."
                    />
                  </div>
                </div>
              )}

              {(!hasLastEvent && selectedTemplate) || hasLastEvent ? (
                <>
                  <div className="flex items-center gap-3">
                    <MaterialSymbol name="label" size="sm" className="text-gray-600 dark:text-gray-400" />
                    <label htmlFor="eventType" className="text-xs font-medium text-gray-700 dark:text-gray-300 whitespace-nowrap w-18">
                      Event Type
                    </label>
                    <Input
                      id="eventType"
                      value={eventType}
                      onChange={(e) => setEventType(e.target.value)}
                      placeholder="e.g., webhook, push, deployment_complete"
                      disabled={isSubmitting}
                      className="flex-1"
                    />
                  </div>

                  <div className="flex-1 flex flex-col">
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
                </>
              ) : null}

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
        <div className="flex items-center justify-end gap-3 px-6 py-5 border-t border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-800">
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