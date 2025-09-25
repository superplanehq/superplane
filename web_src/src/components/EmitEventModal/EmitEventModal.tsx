import { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import Editor from '@monaco-editor/react';
import { Input } from '@/components/Input/input';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Button } from '@/components/Button/button';
import { Select } from '@/components/Select';
import { SuperplaneEvent } from '@/api-client';
import GithubLogo from '@/assets/github-mark.svg';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';

interface EventTemplate {
  name: string;
  description?: string;
  icon?: string;
  image?: string;
  eventType: string;
  nodeType: 'event_source' | 'stage';
  getEventData: () => any;
}

const EVENT_TEMPLATES: EventTemplate[] = [
  {
    name: 'GitHub - Push Event',
    description: 'Event emitted when code is pushed to a GitHub repository',
    image: GithubLogo,
    eventType: 'push',
    nodeType: 'event_source' as const,
    getEventData: () => ({
      ref: "refs/heads/main",
      before: "2364960799e343f8cb594a81b1f34e7219f8254a",
      after: "7fcca06c1b2b2c482df382248610d46cfd789837",
      repository: {
        name: "superplane",
        full_name: "superplanehq/superplane",
        private: false,
        owner: {
          name: "superplanehq",
          email: null,
          login: "superplanehq",
          avatar_url: "https://avatars.githubusercontent.com/u/210748804?v=4",
          gravatar_id: "",
          url: "https://api.github.com/users/superplanehq",
          html_url: "https://github.com/superplanehq",
          type: "Organization",
          user_view_type: "public",
          site_admin: false
        },
        html_url: "https://github.com/superplanehq/superplane",
        description: null,
        fork: false,
        url: "https://api.github.com/repos/superplanehq/superplane",
        created_at: 1746640119,
        updated_at: "2025-09-24T18:47:53Z",
        pushed_at: 1758745245,
        git_url: "git://github.com/superplanehq/superplane.git",
        ssh_url: "git@github.com:superplanehq/superplane.git",
        clone_url: "https://github.com/superplanehq/superplane.git",
        visibility: "public",
        default_branch: "main",
        master_branch: "main",
        organization: "superplanehq",
      },
      pusher: {
        name: "lucaspin",
        email: "lucas@superplane.com"
      },
      organization: {
        login: "superplanehq",
        url: "https://api.github.com/orgs/superplanehq",
        avatar_url: "https://avatars.githubusercontent.com/u/210748804?v=4",
        description: null
      },
      sender: {
        login: "lucaspin",
        avatar_url: "https://avatars.githubusercontent.com/u/12387728?v=4",
        gravatar_id: "",
        url: "https://api.github.com/users/lucaspin",
        html_url: "https://github.com/lucaspin",
        type: "User",
        user_view_type: "public",
        site_admin: false
      },
      created: false,
      deleted: false,
      forced: false,
      base_ref: null,
      compare: "https://github.com/superplanehq/superplane/compare/2364960799e3...7fcca06c1b2b",
      commits: [
        {
          id: "7fcca06c1b2b2c482df382248610d46cfd789837",
          tree_id: "fb692ac9149f575c86374f7cd54e0ba703b03609",
          distinct: true,
          message: "refactor(ui): display only last processed event in event source node (#315)",
          timestamp: "2025-09-24T17:20:44-03:00",
          url: "https://github.com/superplanehq/superplane/commit/7fcca06c1b2b2c482df382248610d46cfd789837",
          author: {
            name: "Lucas Pinheiro",
            email: "lucas@superplane.com",
            username: "lucaspin"
          },
          committer: {
            name: "GitHub",
            email: "noreply@github.com",
            username: "web-flow"
          },
          added: [],
          removed: [
            "pkg/openapi_client/model_status_history.go"
          ],
          modified: [
            "api/swagger/superplane.swagger.json",
            "pkg/grpc/actions/event_sources/create_event_source.go",
            "pkg/grpc/actions/event_sources/describe_event_source.go",
            "pkg/grpc/actions/event_sources/list_event_sources.go",
            "pkg/grpc/actions/events/list_events.go",
            "pkg/grpc/actions/events/list_events_test.go",
            "pkg/grpc/canvas_service.go",
            "pkg/models/event.go",
            "pkg/models/event_source.go",
            "pkg/openapi_client/.openapi-generator/FILES",
            "pkg/openapi_client/api_event.go",
            "pkg/openapi_client/model_superplane_event_source_status.go",
            "pkg/protos/canvases/canvases.pb.go",
            "protos/canvases.proto",
            "web_src/src/api-client/types.gen.ts",
            "web_src/src/hooks/useCanvasData.ts",
            "web_src/src/pages/canvas/components/EventSourceSidebar.tsx",
            "web_src/src/pages/canvas/components/EventStateItem.tsx",
            "web_src/src/pages/canvas/components/nodes/event_source.tsx",
            "web_src/src/pages/canvas/index.tsx"
          ]
        }
      ],
      head_commit: {
        id: "7fcca06c1b2b2c482df382248610d46cfd789837",
        tree_id: "fb692ac9149f575c86374f7cd54e0ba703b03609",
        distinct: true,
        message: "refactor(ui): display only last processed event in event source node (#315)",
        timestamp: "2025-09-24T17:20:44-03:00",
        url: "https://github.com/superplanehq/superplane/commit/7fcca06c1b2b2c482df382248610d46cfd789837",
        author: {
          name: "Lucas Pinheiro",
          email: "lucas@superplane.com",
          username: "lucaspin"
        },
        committer: {
          name: "GitHub",
          email: "noreply@github.com",
          username: "web-flow"
        },
        added: [],
        removed: [
          "pkg/openapi_client/model_status_history.go"
        ],
        modified: [
          "api/swagger/superplane.swagger.json",
          "pkg/grpc/actions/event_sources/create_event_source.go",
          "pkg/grpc/actions/event_sources/describe_event_source.go",
          "pkg/grpc/actions/event_sources/list_event_sources.go",
          "pkg/grpc/actions/events/list_events.go",
          "pkg/grpc/actions/events/list_events_test.go",
          "pkg/grpc/canvas_service.go",
          "pkg/models/event.go",
          "pkg/models/event_source.go",
          "pkg/openapi_client/.openapi-generator/FILES",
          "pkg/openapi_client/api_event.go",
          "pkg/openapi_client/model_superplane_event_source_status.go",
          "pkg/protos/canvases/canvases.pb.go",
          "protos/canvases.proto",
          "web_src/src/api-client/types.gen.ts",
          "web_src/src/hooks/useCanvasData.ts",
          "web_src/src/pages/canvas/components/EventSourceSidebar.tsx",
          "web_src/src/pages/canvas/components/EventStateItem.tsx",
          "web_src/src/pages/canvas/components/nodes/event_source.tsx",
          "web_src/src/pages/canvas/index.tsx"
        ]
      }
    })
  },
  {
    name: 'Semaphore - Pipeline Done Event',
    description: 'Event emitted when a Semaphore CI/CD pipeline completes',
    image: SemaphoreLogo,
    eventType: 'pipeline_done',
    nodeType: 'event_source' as const,
    getEventData: () => ({
      version: "1.0.0",
      organization: {
        name: "superplanehq",
        id: crypto.randomUUID()
      },
      project: {
        name: "superplane",
        id: crypto.randomUUID()
      },
      repository: {
        url: "https://github.com/superplanehq/superplane",
        slug: "superplanehq/superplane"
      },
      revision: {
        sender: {
          login: "lucaspin",
          email: "lucas@superplane.com",
        },
        reference_type: "branch",
        reference: "refs/heads/main",
        pull_request: null,
        commit_sha: "7fcca06c1b2b2c482df382248610d46cfd789837",
        commit_message: "refactor(ui): display only last processed event in event source node (#315)",
        branch: {
          name: "main",
          commit_range: "2364960799e343f8cb594a81b1f34e7219f8254a...7fcca06c1b2b2c482df382248610d46cfd789837"
        }
      },
      workflow: {
        initial_pipeline_id: crypto.randomUUID(),
        id: crypto.randomUUID(),
        created_at: new Date().toISOString()
      },
      pipeline: {
        yaml_file_name: "semaphore.yml",
        working_directory: ".semaphore",
        stopping_at: new Date().toISOString(),
        state: "done",
        running_at: new Date().toISOString(),
        result_reason: "",
        result: "passed",
        queuing_at: new Date().toISOString(),
        pending_at: new Date().toISOString(),
        name: "Pipeline",
        id: crypto.randomUUID(),
        error_description: "",
        done_at: new Date().toISOString(),
        created_at: new Date().toISOString()
      },
      blocks: [
        {
          state: "done",
          result_reason: "",
          result: "passed",
          name: "List & Test & Build",
          jobs: [
            {
              status: "finished",
              result: "passed",
              name: "Test",
              index: 1,
              id: crypto.randomUUID()
            },
            {
              status: "finished",
              result: "passed",
              name: "Build",
              index: 2,
              id: crypto.randomUUID()
            },
            {
              status: "finished",
              result: "passed",
              name: "Lint",
              index: 0,
              id: crypto.randomUUID()
            }
          ]
        }
      ]
    })
  },
  {
    name: 'Custom WebHook',
    description: 'Create your own custom webhook event with custom data',
    icon: 'webhook',
    eventType: 'custom',
    nodeType: 'event_source' as const,
    getEventData: () => ({})
  },
  {
    name: 'SuperPlane - Execution Finished Event',
    description: 'Event emitted when a SuperPlane stage execution completes',
    icon: 'task_alt',
    eventType: 'execution_finished',
    nodeType: 'stage' as const,
    getEventData: () => ({
      type: "execution_finished",
      stage: {
        id: crypto.randomUUID()
      },
      execution: {
        created_at: new Date().toISOString(),
        finished_at: new Date().toISOString(),
        id: crypto.randomUUID(),
        result: "passed",
        result_message: "",
        result_reason: ""
      },
    })
  }
];

interface EmitEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  sourceName: string;
  nodeType: 'event_source' | 'stage';
  loadLastEvent: () => Promise<SuperplaneEvent | null>;
  onCancel?: () => void;
  onSubmit: (eventType: string, eventData: any) => Promise<void>;
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