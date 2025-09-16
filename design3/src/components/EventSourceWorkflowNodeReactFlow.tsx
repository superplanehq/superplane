import { useState, useCallback, useEffect } from 'react';
import { Handle, Position } from '@xyflow/react';
import clsx from 'clsx';
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol';
import { Button } from './lib/Button/button';
import { Input } from './lib/Input/input';
import { Textarea } from './lib/Textarea/textarea';
import { Field, Label } from './lib/Fieldset/fieldset';
import { Link } from './lib/Link/link';
import { BadgeButton, Badge } from './lib/Badge/badge';
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from './lib/Dialog/dialog';
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem, DropdownLabel } from './lib/Dropdown/dropdown';
import { ControlledTabs, type Tab } from './lib/Tabs/tabs';
import Tippy from '@tippyjs/react';
import { Text } from './lib/Text/text';
import { EmptyState } from './lib/EmptyState/empty-state';

export interface EventSourceWorkflowNodeReactFlowData {
  id: string;
  title: string;
  description?: string;
  cluster?: string;
  events: Array<{
    id: string;
    url: string;
    type?: string;
    enabled?: boolean;
  }>;
  icon?: string;
  eventSourceType?: 'semaphore' | 'webhook' | 'http' | 'schedule';
  // For scheduled event sources, show the upcoming execution time
  nextRunTime?: string; // ISO string
  // Optional cron schedule (for schedule type)
  scheduleCron?: string;
  // Optional rich text payload (stored as HTML)
  payload?: string;
  // Whether the schedule is enabled (for schedule type)
  scheduleEnabled?: boolean;
  selected?: boolean;
  isEditMode?: boolean;
}

interface EventSourceWorkflowNodeReactFlowProps {
  data: EventSourceWorkflowNodeReactFlowData;
  selected?: boolean;
}

export function EventSourceWorkflowNodeReactFlow({ 
  data, 
  selected = false 
}: EventSourceWorkflowNodeReactFlowProps) {
  const [isEditMode, setIsEditMode] = useState(data.isEditMode || false);
  const [editData, setEditData] = useState({
    title: data.title,
    description: data.description || '',
    cluster: data.cluster || '',
    events: [...data.events],
    scheduleCron: data.scheduleCron || (data.eventSourceType === 'schedule' ? '0 0 * * 1' : ''),
    payload: data.payload || '',
    scheduleEnabled: typeof data.scheduleEnabled === 'boolean' ? data.scheduleEnabled : true
  });
  // Inline editing state (align with WorkflowNodeAccordion pattern)
  const [editingField, setEditingField] = useState<null | 'title' | 'description'>(null);
  const [tempTitle, setTempTitle] = useState<string>(editData.title);
  const [tempDescription, setTempDescription] = useState<string>(editData.description);

  // Helper to format next run label for schedule-based event sources
  const getNextRunLabel = useCallback(() => {
    if (!data.nextRunTime) return 'Next run: not scheduled';
    const now = new Date();
    const next = new Date(data.nextRunTime);
    const diffMs = next.getTime() - now.getTime();
    if (diffMs <= 0) return 'Next run: due now';
    const mins = Math.round(diffMs / 60000);
    if (mins < 60) return `Next run: in ${mins}m`;
    const hours = Math.floor(mins / 60);
    const remMins = mins % 60;
    return `Next run: in ${hours}h ${remMins}m`;
  }, [data.nextRunTime]);
  
  // Local state for events to persist changes in read-only mode  
  // Enhanced events with status indicators for preview mode
  const [displayEvents, setDisplayEvents] = useState(
    data.events.map((event, index) => ({
      ...event,
      status: (index === 0 ? 'pending' : index === 1 ? 'discarded' : 'forwarded') as 'pending' | 'discarded' | 'forwarded',
      timestamp: new Date(Date.now() - (index * 60 * 1000)).toISOString(),
      processingTime: index === 2 ? 245 : undefined
    }))
  );
  
  // Check URL parameter for inline integration mode
  const urlParams = new URLSearchParams(window.location.search);
  const inlineIntegration = urlParams.get('inlineIntegration') === 'true';
  
  // Check URL parameter for event filters version (defaults to 1)
  const eventFiltersVersion = urlParams.get('eventFiltersVersion') === '2' ? 2 : 1;
  
  // Check URL parameter for compact event display (defaults to true)
  const compactEvent = urlParams.get('compactEvent') !== 'false';
  
  // Integration modal state
  const [showIntegrationModal, setShowIntegrationModal] = useState(false);
  const [integrationData, setIntegrationData] = useState({
    name: '',
    orgUrl: '',
    apiToken: {
      secretName: '',
      secretKey: ''
    }
  });
  
  // Inline integration form state
  const [inlineIntegrationData, setInlineIntegrationData] = useState({
    orgUrl: '',
    apiToken: ''
  });
  
  // Secrets management
  const [secrets, setSecrets] = useState<Array<{
    id: string;
    name: string;
    keys: Array<{ key: string; value: string; }>;
  }>>([
    // Mock existing secret
    {
      id: 'secret-1',
      name: 'my semaphore org secrets',
      keys: [{ key: 'API_TOKEN', value: 'hidden' }]
    }
  ]);
  const [newSecret, setNewSecret] = useState({
    name: '',
    key: '',
    value: ''
  });
  
  // Derived: currently selected existing secret (if any)
  const selectedExistingSecret = secrets.find(
    (s) => s.name === integrationData.apiToken.secretName
  );
  
  // API Token tabs state
  const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('new');
  const [newSecretToken, setNewSecretToken] = useState('');
  const [showGitHubPatInfo, setShowGitHubPatInfo] = useState(false);
  
  // Selected integration state
  const [selectedIntegration, setSelectedIntegration] = useState<string | null>(null);
  
  // Sidebar state for event details
  const [showSidebar, setShowSidebar] = useState(false);
  
  // Webhook icon picker state
  const [currentIcon, setCurrentIcon] = useState<string>(data.icon || 'webhook');
  const [showIconModal, setShowIconModal] = useState(false);
  const [iconSearchQuery, setIconSearchQuery] = useState('');
  
  // Available Material Symbol icons for webhook nodes
  const materialIconOptions = [
    { name: 'webhook', label: 'Webhook', type: 'material' },
    { name: 'api', label: 'API', type: 'material' },
    { name: 'integration_instructions', label: 'Integration', type: 'material' },
    { name: 'http', label: 'HTTP', type: 'material' },
    { name: 'sync', label: 'Sync', type: 'material' },
    { name: 'cloud', label: 'Cloud', type: 'material' },
    { name: 'settings', label: 'Settings', type: 'material' },
    { name: 'code', label: 'Code', type: 'material' },
    { name: 'build', label: 'Build', type: 'material' },
    { name: 'send', label: 'Send', type: 'material' },
    { name: 'link', label: 'Link', type: 'material' },
    { name: 'bolt', label: 'Bolt', type: 'material' },
  ];

  // Available DevOps tool icons (similar to component sidebar)
  const devopsIconOptions = [
    { name: 'docker', label: 'Docker', icon: '/images/docker-logo.svg', type: 'devops' },
    { name: 'kubernetes', label: 'Kubernetes', icon: '/images/kubernetes-logo.svg', type: 'devops' },
    { name: 'terraform', label: 'Terraform', icon: '/images/terraform-logo.svg', type: 'devops' },
    { name: 'aws', label: 'AWS', icon: '/images/aws-logo.svg', type: 'devops' },
    { name: 'git', label: 'Git', icon: '/images/git-logo.svg', type: 'devops' },
    { name: 'github', label: 'GitHub', icon: '/images/github-logo.svg', type: 'devops' },
    { name: 'npm', label: 'NPM', icon: '/images/npm-logo.svg', type: 'devops' },
    { name: 'python', label: 'Python', icon: '/images/python-logo.svg', type: 'devops' },
    { name: 'helm', label: 'Helm', icon: '/images/helm-logo.svg', type: 'devops' },
    { name: 'ansible', label: 'Ansible', icon: '/images/ansible-logo.svg', type: 'devops' },
    { name: 'sonarqube', label: 'SonarQube', icon: '/images/sonarqube-logo.svg', type: 'devops' },
    { name: 'semaphore', label: 'Semaphore', icon: '/images/semaphore-logo-sign-black.svg', type: 'devops' },
  ];
  
  // Combined icon options
  const allIconOptions = [...materialIconOptions, ...devopsIconOptions];
  
  // Filter icons based on search query
  const filteredMaterialIcons = materialIconOptions.filter(icon => 
    icon.name.toLowerCase().includes(iconSearchQuery.toLowerCase()) ||
    icon.label.toLowerCase().includes(iconSearchQuery.toLowerCase())
  );
  
  const filteredDevopsIcons = devopsIconOptions.filter(icon => 
    icon.name.toLowerCase().includes(iconSearchQuery.toLowerCase()) ||
    icon.label.toLowerCase().includes(iconSearchQuery.toLowerCase())
  );
  
  // Handle icon change for webhook nodes
  const handleIconChange = useCallback((newIcon: string) => {
    setCurrentIcon(newIcon);
    setShowIconModal(false);
    setIconSearchQuery(''); // Reset search when selecting an icon
    // Update the node data immediately so it shows in both edit and read mode
    if (data) {
      data.icon = newIcon;
    }
  }, [data]);

  // Handle modal close to reset search
  const handleIconModalClose = useCallback(() => {
    setShowIconModal(false);
    setIconSearchQuery(''); // Reset search when closing modal
  }, []);

  // Pretty-print payload JSON (fallback to raw text)
  const getPayloadPreview = useCallback(() => {
    const placeholderObj = {
      sha: '15990b',
      name: 'BUG-1982: Connections',
      image: 'docker-196',
    };
    const prettyPlaceholder = JSON.stringify(placeholderObj, null, 2);
    const raw = (editData.payload || '').trim();
    if (!raw) return prettyPlaceholder;
    try {
      const parsed = JSON.parse(raw);
      return JSON.stringify(parsed, null, 2);
    } catch {
      // If it's not valid JSON, try to normalize quotes; otherwise return as-is
      return raw;
    }
  }, [editData.payload]);

  // Helper function to render icon (Material Symbol or DevOps image)
  const renderIcon = useCallback((iconName: string, size: 'sm' | 'md' | 'lg' = 'lg') => {
    const devopsIcon = devopsIconOptions.find(icon => icon.name === iconName);
    if (devopsIcon) {
      const iconSize = size === 'sm' ? 16 : size === 'md' ? 20 : 24;
      return <img width={iconSize} height={iconSize} src={devopsIcon.icon} alt={devopsIcon.label} className="flex-shrink-0" />;
    }
    // Default to Material Symbol
    return <MaterialSymbol name={iconName} size={size} className="text-gray-700 dark:text-zinc-300" />;
  }, [devopsIconOptions]);
  
  // No state needed for hover-triggered popovers
  
  // Mock filters data
  const appliedFilters = [
    {
      id: 'filter-1',
      type: 'Branch',
      value: 'Head',
      operator: 'contains'
    },
    {
      id: 'filter-2', 
      type: 'Event Type',
      value: 'Data',
      operator: 'contains'
    }
  ];
  
  // Available integrations (mock data + created ones)
  const [availableIntegrations, setAvailableIntegrations] = useState<Array<{
    id: string;
    name: string;
    orgUrl: string;
    secretName: string;
  }>>([]);
  
  // Define tabs for API Token section
  const apiTokenTabs: Tab[] = [
    {
      id: 'new',
      label: 'Create new secret',
    },
    {
      id: 'existing',
      label: 'Import from existing secret',
    }
  ];


  const handleSave = useCallback(() => {
    // Save as Draft: preserve current editData.events in displayEvents with enhanced properties
    const enhancedEvents = editData.events.map((event, index) => ({
      ...event,
      status: (index === 0 ? 'pending' : index === 1 ? 'discarded' : 'forwarded') as 'pending' | 'discarded' | 'forwarded',
      timestamp: new Date(Date.now() - (index * 60 * 1000)).toISOString(),
      processingTime: index === 2 ? 245 : undefined
    }));
    setDisplayEvents(enhancedEvents);
    
    // Here you would typically save the data to your backend or state management
    setIsEditMode(false);
  }, [editData.events]);

  const handleSaveAndCommit = useCallback(() => {
    // Save & Commit: transition to read-mode without events to show noEvents variant
    setIsEditMode(false);
    
    // Clear events to demonstrate the noEvents variant
    setDisplayEvents([]);
    setEditData(prev => ({
      ...prev,
      events: []
    }));
    
    // Here you would typically commit the changes to your backend or state management
    console.log('Save & Commit: Transitioning to read-mode without events');
  }, []);

  const handleCancel = useCallback(() => {
    setEditData({
      title: data.title,
      description: data.description || '',
      cluster: data.cluster || '',
      events: [...data.events],
      scheduleCron: data.scheduleCron || (data.eventSourceType === 'schedule' ? '0 0 * * 1' : ''),
      payload: data.payload || '',
      scheduleEnabled: typeof data.scheduleEnabled === 'boolean' ? data.scheduleEnabled : true
    });
    setIsEditMode(false);
  }, [data]);


  const handleCreateIntegration = useCallback(() => {
    setShowIntegrationModal(true);
  }, []);


  const handleSaveIntegration = useCallback(() => {
    // Create new integration
    const newIntegration = {
      id: `integration-${Date.now()}`,
      name: integrationData.name,
      orgUrl: integrationData.orgUrl,
      secretName: integrationData.apiToken.secretName
    };
    
    // If creating a new secret, add it to secrets
    if (apiTokenTab === 'new' && newSecretToken) {
      const newSecret = {
        id: `secret-${Date.now()}`,
        name: `${integrationData.name} Secret`,
        keys: [{ key: 'API_TOKEN', value: newSecretToken }]
      };
      setSecrets(prev => [...prev, newSecret]);
      newIntegration.secretName = newSecret.name;
    }
    
    // Add the integration to available integrations and select it
    setAvailableIntegrations(prev => [...prev, newIntegration]);
    setSelectedIntegration(newIntegration.id);
    
    // Close modal and reset form
    setShowIntegrationModal(false);
    setIntegrationData({
      name: '',
      orgUrl: '',
      apiToken: { secretName: '', secretKey: '' }
    });
    setNewSecretToken('');
    setApiTokenTab('new');
  }, [integrationData, apiTokenTab, newSecretToken]);

  const truncateUrl = (url: string, maxLength: number = 40) => {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  };


  // Watch for node selection to open sidebar
  useEffect(() => {
    console.log('Selected prop changed:', selected);
    if (selected && !isEditMode) {
      console.log('Node is selected and not in edit mode, opening sidebar');
      setShowSidebar(true);
    } else if (!selected) {
      console.log('Node is not selected, closing sidebar');
      setShowSidebar(false);
    }
  }, [selected, isEditMode]);

  // Handle sidebar close - also deselect the node
  const handleSidebarClose = useCallback(() => {
    console.log('Closing sidebar');
    setShowSidebar(false);
    // Note: We don't deselect the node here as React Flow manages that
  }, []);

  // (removed) applyFormat toolbar helper not needed; payload is preview-only

  const handleEdit = useCallback(() => {
    setIsEditMode(true);
  }, []);

  const handleFormSave = useCallback(() => {
    // Persist to data so preview reflects changes immediately
    if (data) {
      data.title = editData.title;
      data.description = editData.description;
      data.scheduleCron = editData.scheduleCron;
      data.payload = editData.payload;
      data.scheduleEnabled = editData.scheduleEnabled;
    }
    handleSave();
  }, [data, editData, handleSave]);

  const handleFormCancel = useCallback(() => {
    setEditData({
      title: data.title,
      description: data.description || '',
      cluster: data.cluster || '',
      events: [...data.events],
      scheduleCron: data.scheduleCron || (data.eventSourceType === 'schedule' ? '*/30 * * * *' : ''),
      payload: data.payload || '',
      scheduleEnabled: typeof data.scheduleEnabled === 'boolean' ? data.scheduleEnabled : true
    });
    setTempTitle(data.title);
    setTempDescription(data.description || '');
    setEditingField(null);
    handleCancel();
  }, [data, handleCancel]);

  // Lightweight helpers for cron description and UTC formatting
  const describeCron = useCallback((expr: string) => {
    const trimmed = (expr || '').trim();
    switch (trimmed) {
      case '* * * * *':
        return 'Every minute';
      case '*/5 * * * *':
        return 'Every 5 minutes';
      case '*/15 * * * *':
        return 'Every 15 minutes';
      case '0 * * * *':
        return 'At minute 0, every hour';
      case '0 0 * * *':
        return 'At 12:00 AM, every day';
      case '0 0 * * 1':
        return 'At 12:00 AM, every Monday';
      default:
        return 'Custom schedule';
    }
  }, []);

  const formatUtc = useCallback((iso?: string) => {
    if (!iso) return '—';
    try {
      const d = new Date(iso);
      const pad = (n: number) => String(n).padStart(2, '0');
      return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())} UTC`;
    } catch {
      return '—';
    }
  }, []);

  // Preview Mode
  console.log('EventSourceWorkflowNode render:', { isEditMode, showSidebar, selected }); // Debug log
  
  if (isEditMode) {
    // Edit Mode UI
    return (
      <div 
        className={clsx(
          'bg-white dark:bg-zinc-800 rounded-lg border-2 relative transition-all duration-200 min-w-[340px]',
          'border-blue-600 dark:border-zinc-200 ring-2 ring-blue-200 dark:ring-white'
        )}
        style={{ width: 340 }}
        role="form"
      >
        {/* Action buttons bar in edit mode */}
        <div 
          className="action-buttons absolute -top-14 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-xs rounded-lg p-1 border border-gray-200 dark:border-zinc-600 z-50"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Code first to match other components */}
          <Button type="button" plain className="flex items-center gap-2">
            <MaterialSymbol name="code" size="md"/>
            Code
          </Button>
          {/* Save (plain style to match) */}
          <Button plain onClick={handleFormSave} className="flex items-center gap-2">
            <MaterialSymbol name="save" size="md"/>
            Save
          </Button>
          {/* Cancel */}
          <Button plain onClick={handleFormCancel} className="flex items-center gap-2">
            <MaterialSymbol name="close" size="md"/>
            Cancel
          </Button>

          {/* More options dropdown */}
          <Dropdown>
            <DropdownButton plain className='flex items-center gap-2 !pr-1'>
              <MaterialSymbol name="more_vert" size="md"/>
            </DropdownButton>
            <DropdownMenu anchor="bottom start">
              <DropdownItem className='flex items-center gap-2'>
                <MaterialSymbol name="play_arrow" size="md"/>
                <DropdownLabel>Run</DropdownLabel>
              </DropdownItem>
              <DropdownItem className='flex items-center gap-2'>
                <MaterialSymbol name="tune" size="md"/>
                <DropdownLabel>Advanced configuration</DropdownLabel>
              </DropdownItem>
              <DropdownItem className='flex items-center gap-2'>
                <MaterialSymbol name="menu_book" size="md"/>
                <DropdownLabel>Documentation</DropdownLabel>
              </DropdownItem>
              <DropdownItem className='flex items-center gap-2 text-red-600 dark:text-red-200' color='red'>
                <MaterialSymbol name="delete" size="md"/>
                <DropdownLabel>Delete</DropdownLabel>
              </DropdownItem>
            </DropdownMenu>
          </Dropdown>
        </div>
        {/* Header with inline editable Title */}
        <div className="p-4 border-b border-gray-200 dark:border-zinc-700 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className='flex items-center content-center bg-zinc-100 dark:bg-zinc-700 rounded-md w-10 h-10'>
              {data.eventSourceType === 'schedule' ? (
                <MaterialSymbol name="schedule" size="lg" className="text-gray-700 dark:text-zinc-300 m-auto" />
              ) : (
                <MaterialSymbol name="sensors" size="lg" className="text-gray-700 dark:text-zinc-300 m-auto" />
              )}
            </div>
            <div className="flex flex-col">
              {/* Inline editable title */}
              {editingField === 'title' ? (
                <Input
                  value={tempTitle}
                  onChange={(e) => setTempTitle(e.target.value)}
                  onKeyDown={(e: any) => {
                    if (e.key === 'Enter') {
                      setEditData({ ...editData, title: tempTitle });
                      setEditingField(null);
                    } else if (e.key === 'Escape') {
                      setTempTitle(editData.title);
                      setEditingField(null);
                    }
                  }}
                  onBlur={() => { setEditData({ ...editData, title: tempTitle }); setEditingField(null); }}
                  className="font-semibold text-gray-900 dark:text-white"
                  autoFocus
                />
              ) : (
                <div className="group relative">
                  <div className="flex items-center">
                    <h3 className="font-semibold text-gray-900 dark:text-white mr-2">{editData.title}</h3>
                    <button
                      type="button"
                      onClick={() => setEditingField('title')}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
                      aria-label="Edit title"
                    >
                      <MaterialSymbol name="edit" size="sm" />
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
          <div />
        </div>

        {/* Body Form */}
        <div className="p-4 space-y-4">
          {/* Inline editable description */}
          {editingField === 'description' ? (
            <div>
              <Textarea
                value={tempDescription}
                onChange={(e: any) => setTempDescription(e.target.value)}
                onKeyDown={(e: any) => {
                  if (e.key === 'Escape') {
                    setTempDescription(editData.description);
                    setEditingField(null);
                  }
                }}
                onBlur={() => { setEditData({ ...editData, description: tempDescription }); setEditingField(null); }}
                rows={3}
              />
            </div>
          ) : (
            <div
              className="group relative cursor-text"
              onClick={() => setEditingField('description')}
              role="button"
              aria-label="Edit description"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === 'Enter') setEditingField('description');
              }}
            >
              <div className="text-sm text-zinc-700 dark:text-zinc-300 whitespace-pre-wrap">
                {editData.description || 'Add a description...'}
              </div>
            </div>
          )}
          {/* Removed standalone Name/Description fields to keep inline editing UX */}

          {data.eventSourceType === 'schedule' && (
            <Field>
              <div className="flex items-center justify-between">
                <Label>Schedule</Label>
                <button
                  type="button"
                  onClick={() => setEditData({ ...editData, scheduleEnabled: !editData.scheduleEnabled })}
                  className="ml-2 cursor-pointer"
                  aria-label="Toggle schedule enabled"
                >
                  <Badge color={editData.scheduleEnabled ? 'blue' : 'zinc'} className='mt-1'>
                    {editData.scheduleEnabled ? 'Enabled' : 'Paused'}
                  </Badge>
                </button>
              </div>
              <div className="mb-1 mt-2 text-xs text-zinc-600 dark:text-zinc-400">
                For help with Crontab syntax, visit
                {' '}<a className="text-blue-600 dark:text-blue-400" href="https://crontab.guru/" target="_blank" rel="noopener">Crontab Guru</a>.
              </div>
              <Input
                value={editData.scheduleCron}
                onChange={(e) => setEditData({ ...editData, scheduleCron: e.target.value })}
                placeholder="e.g. */15 * * * *"
                className="font-mono text-xs"
              />
              <p className="text-xs mt-1 text-zinc-700 dark:text-zinc-300">
                Translates to: <span className="font-semibold">{describeCron(editData.scheduleCron)}</span>
              </p>
              <p className="text-xs text-zinc-700 dark:text-zinc-300">
                Next scheduled for: <span>{formatUtc(data.nextRunTime)}</span>
              </p>
            </Field>
          )}

          <Field>
            <div className="flex items-center justify-between">
              <Label>Payload</Label>
              <Button
                type="button"
                plain
                className="!p-1"
                aria-label="Edit payload"
              >
                <MaterialSymbol name="edit" size="sm" />
              </Button>
            </div>
            <div className="border border-gray-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-900/40">
              <pre className="p-3 font-mono text-xs text-zinc-800 dark:text-zinc-200 whitespace-pre-wrap">
{getPayloadPreview()}
              </pre>
            </div>
            <div className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">Payload preview (read-only).</div>
          </Field>

          {/* Add new schedule link at bottom (no-op) */}
          <div className='flex items-center justify-between w-full mt-3'>
            <div className='flex items-center gap-1'>
              <Link 
                href="#" 
                className="text-xs text-blue-700 dark:text-blue-400 flex items-center gap-1"
                onClick={(e) => {
                  e.preventDefault();
                }}
              >
                <MaterialSymbol name="add" size="sm"/>
                <span>Add new schedule</span>
              </Link>
            </div>
          </div>
        </div>

        {/* Handles */}
        <Handle type="target" position={Position.Left} className="w-3 h-3 !bg-primary-500 !border-2 !border-white" />
        <Handle type="source" position={Position.Right} className="w-3 h-3 !bg-primary-500 !border-2 !border-white" />
      </div>
    );
  }

  if (!isEditMode) {
    console.log('Rendering preview mode'); // Debug log
    return (
      <>
        <div 
          className={clsx(
            'bg-white dark:bg-zinc-800 rounded-lg border-2 relative transition-all duration-200 hover:shadow-lg hover:border-blue-300 dark:hover:border-blue-600 min-w-[320px] cursor-pointer',
            selected ? 'border-blue-600 dark:border-zinc-200 ring-2 ring-blue-200 dark:ring-white' : 'border-gray-200 dark:border-zinc-700'
          )}
          style={{ width: 320, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
          role="article"
        >
        {/* Action buttons when selected */}
        {selected && (
          <div 
            className="action-buttons absolute -top-14 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-xs rounded-lg p-1 border border-gray-200 dark:border-zinc-600 z-50"
            onClick={(e) => e.stopPropagation()}
          >
            <Button type="button" plain className="flex items-center gap-2" onClick={handleEdit}>
              <MaterialSymbol name="edit" size="md"/>
              Edit
            </Button>
          </div>
        )}
        
        
        {/* Header */}
        <div className="flex flex-col p-4 border-b border-gray-200 dark:border-zinc-700">
          <div className='flex items-start flex-row justify-between w-full'>

            <div className="flex items-center flex-grow-1 gap-3">
              {data.eventSourceType === 'webhook' ? (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowIconModal(true);
                  }}
                  className="flex items-center content-center bg-zinc-100 dark:bg-zinc-700 rounded-md w-10 h-10 hover:bg-zinc-200 dark:hover:bg-zinc-600 transition-colors cursor-pointer"
                  title="Click to change icon"
                >
                  <div className="m-auto">
                    {renderIcon(data.icon || 'webhook')}
                  </div>
                </button>
              ) : (
                <div className='flex items-center content-center bg-zinc-100 dark:bg-zinc-700 rounded-md w-10 h-10'>
                  {data.eventSourceType === 'http' ? (
                    <MaterialSymbol name="rocket_launch" size="lg" className="text-gray-700 dark:text-zinc-300 m-auto" />
                  ) : data.eventSourceType === 'schedule' ? (
                    <MaterialSymbol name="schedule" size="lg" className="text-gray-700 dark:text-zinc-300 m-auto" />
                  ) : data.icon === 'semaphore' ? (
                    <img width={24} height={24} className='m-auto' src='/images/semaphore-logo-sign-black.svg' alt="Semaphore" />
                  ) : data.icon === 'github' ? (
                    <img width={24} height={24} className='m-auto' src='/images/github-logo.svg' alt="GitHub" />
                  ) : (
                    <img width={24} height={24} className='m-auto' src='https://upload.wikimedia.org/wikipedia/commons/3/39/Kubernetes_logo_without_workmark.svg' alt="Kubernetes" />
                  )}
                </div>
              )}
           
              <h3 className="text-md font-semibold text-gray-900 dark:text-white truncate">
                {data.title}
              </h3>
              
           
           
          </div>
           {/* Configuration indicator - only show when properly configured (no events but saved) */}
           <span className='hidden'>
           <BadgeButton color='green'>
            <MaterialSymbol name="check" size="sm" />
           </BadgeButton>
          </span>
          </div>
          <div className='flex items-center gap-3 mt-1 text-blue-600 dark:text-blue-300 mt-4'>
              {data.eventSourceType === 'schedule' ? (
                <BadgeButton 
                  color='zinc' 
                  href='#' 
                  className='!text-xs'
                >
                  <MaterialSymbol name="schedule" size="md"/> {getNextRunLabel()}
                </BadgeButton>
              ) : (
                <Tippy
                  content={
                    <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[250px]">
                      <div className="text-sm font-medium text-zinc-900 dark:text-white mb-3">
                        Project Configuration
                      </div>
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <span className="text-sm text-zinc-600 dark:text-zinc-400">Project:</span>
                          <span className="text-sm font-mono text-zinc-800 dark:text-zinc-200 bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded">
                            semaphore-project
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-sm text-zinc-600 dark:text-zinc-400">Organization:</span>
                          <span className="text-sm text-zinc-700 dark:text-zinc-300">
                            {data.icon === 'semaphore' ? 'Semaphore CI' : data.icon === 'github' ? 'GitHub' : 'External Service'}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-sm text-zinc-600 dark:text-zinc-400">Status:</span>
                          <div className="flex items-center gap-2">
                            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                            <span className="text-xs text-green-600 dark:text-green-400">Connected</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  }
                  interactive={true}
                  placement="bottom"
                  trigger="mouseenter"
                  delay={[200, 100]}
                  className="z-50"
                >
                  <BadgeButton 
                    color='zinc' 
                    href='#' 
                    className='!text-xs'
                  >
                    <MaterialSymbol name="assignment" size="md"/> semaphore-project
                  </BadgeButton>
                </Tippy>
              )}
              
              {/* Event Filters Display - Version 1 (Default) */}
              {eventFiltersVersion === 1 && (
                appliedFilters.length > 0 ? (
                <Tippy
                  content={
                    <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[260px]">
                      <div className="space-y-2">
                        <div className="flex items-start gap-2">
                          <MaterialSymbol name="schedule" size="sm" className="text-zinc-500 dark:text-zinc-400" />
                          <span className="text-xs text-zinc-700 dark:text-zinc-300">At 12:00 AM, every Monday</span>
                        </div>
                        <div className="flex items-start gap-2">
                          <MaterialSymbol name="schedule" size="sm" className="text-zinc-500 dark:text-zinc-400" />
                          <span className="text-xs text-zinc-700 dark:text-zinc-300">At minute 0, every hour</span>
                        </div>
                      </div>
                    </div>
                  }
                  interactive={true}
                  placement="top"
                  trigger="mouseenter"
                  delay={[200, 100]}
                  className="z-50"
                >
                  <BadgeButton 
                    color='zinc' 
                    href='#' 
                    className='!text-xs'
                  >
                    <MaterialSymbol name="schedule" size="md"/>
                    {appliedFilters.length} Schedules
                  </BadgeButton>
                </Tippy>
                ) : (
                  <BadgeButton 
                    color='zinc' 
                    href='#' 
                    className='!text-xs'
                  >
                    <MaterialSymbol name="schedule" size="md"/>
                    No schedules
                  </BadgeButton>
                )
              )}

              {/* Event Filters Display - Version 2 (Enhanced Card Layout) */}
              {eventFiltersVersion === 2 && (
                <div className="flex flex-wrap gap-2 max-w-full">
                  {appliedFilters.map((filter) => (
                    <Tippy
                      key={filter.id}
                      content={
                        <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-xl shadow-lg p-4 min-w-[280px]">
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex items-center gap-2">
                              <div className="w-8 h-8 bg-gradient-to-br from-blue-100 to-purple-100 dark:from-blue-900/40 dark:to-purple-900/40 rounded-lg flex items-center justify-center">
                                <MaterialSymbol name="filter_alt" size="sm" className="text-blue-600 dark:text-blue-400" />
                              </div>
                              <div>
                                <div className="text-sm font-semibold text-zinc-900 dark:text-white">
                                  {filter.type} Filter
                                </div>
                                <div className="text-xs text-zinc-500 dark:text-zinc-400">
                                  Active condition
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-1">
                              <button className="p-1 hover:bg-zinc-100 dark:hover:bg-zinc-700 rounded transition-colors">
                                <MaterialSymbol name="edit" size="sm" className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300" />
                              </button>
                              <button className="p-1 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors">
                                <MaterialSymbol name="close" size="sm" className="text-zinc-400 hover:text-red-600 dark:hover:text-red-400" />
                              </button>
                            </div>
                          </div>
                          
                          <div className="space-y-2 mb-3">
                            <div className="flex items-center justify-between">
                              <span className="text-xs text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">Condition</span>
                              <div className="flex items-center gap-2">
                                <span className="text-xs font-mono bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-700 dark:text-zinc-300">
                                  {filter.operator}
                                </span>
                                <span className="text-xs font-mono bg-blue-100 dark:bg-blue-900/40 px-2 py-1 rounded text-blue-700 dark:text-blue-300 font-semibold">
                                  "{filter.value}"
                                </span>
                              </div>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-xs text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">Status</span>
                              <div className="flex items-center gap-1">
                                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                                <span className="text-xs text-green-600 dark:text-green-400 font-medium">Active</span>
                              </div>
                            </div>
                          </div>
                          
                          <div className="pt-2 border-t border-zinc-200 dark:border-zinc-700">
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              Events matching this filter will trigger the workflow
                            </div>
                          </div>
                        </div>
                      }
                      interactive={true}
                      placement="bottom"
                      trigger="mouseenter"
                      delay={[200, 100]}
                      className="z-50"
                      maxWidth={320}
                    >
                      <div className="group flex items-center gap-2 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 hover:from-blue-100 hover:to-purple-100 dark:hover:from-blue-900/30 dark:hover:to-purple-900/30 text-blue-800 dark:text-blue-200 px-3 py-2 rounded-lg text-xs border border-blue-200/50 dark:border-blue-800/50 hover:border-blue-300 dark:hover:border-blue-700 transition-all duration-200 cursor-pointer shadow-sm hover:shadow-md min-w-0">
                        <div className="w-4 h-4 bg-blue-600 dark:bg-blue-400 rounded-full flex items-center justify-center flex-shrink-0">
                          <MaterialSymbol name="filter_alt" size={12} className="text-white dark:text-blue-900" />
                        </div>
                        <div className="flex items-center gap-1 min-w-0">
                          <span className="font-semibold truncate">{filter.type}</span>
                          <span className="text-blue-600 dark:text-blue-400 flex-shrink-0">:</span>
                          <span className="font-mono font-medium truncate max-w-20" title={filter.value}>
                            {filter.value.length > 8 ? `${filter.value.substring(0, 8)}...` : filter.value}
                          </span>
                        </div>
                        <MaterialSymbol 
                          name="info" 
                          size="sm" 
                          className="text-blue-500 dark:text-blue-400 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0" 
                        />
                      </div>
                    </Tippy>
                  ))}
                  
                  {/* Add Filter Button */}
                  <Tippy
                    content={
                      <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-3">
                        <div className="text-xs font-medium text-zinc-900 dark:text-white">
                          Add Event Filter
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                          Filter events based on branches, types, or custom conditions
                        </div>
                      </div>
                    }
                    placement="bottom"
                    trigger="mouseenter"
                    delay={[200, 100]}
                  >
                    <button className="flex items-center gap-1 bg-zinc-100 hover:bg-zinc-200 dark:bg-zinc-800 dark:hover:bg-zinc-700 text-zinc-600 dark:text-zinc-400 px-3 py-2 rounded-lg text-xs border border-dashed border-zinc-300 dark:border-zinc-600 hover:border-zinc-400 dark:hover:border-zinc-500 transition-all duration-200">
                      <MaterialSymbol name="add" size="sm" />
                      <span className="font-medium">Add Filter</span>
                    </button>
                  </Tippy>
                </div>
              )}
              
            </div>
        </div>

        {/* Cluster Section */}
        {data.cluster && (
          <div className="px-4 py-3 pb-0 hidden">
            <div className="text-blue-600 dark:text-blue-400 font-medium">
              {data.cluster}
            </div>
          </div>
        )}

        {/* Description (if present) */}
        {data.description && (
          <div className="px-4 pt-3 text-sm text-zinc-700 dark:text-zinc-300">
            {data.description}
          </div>
        )}

        {/* Events Section */}
        <div className="">
          
          
          {displayEvents.length > 0 ? (
            /* Events list with compact/expanded mode support */
            <div className={compactEvent ? "space-y-1 p-3" : "space-y-2 p-4"}>
              <div className='flex items-center justify-between w-full mb-3'>
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide">
                 LATEST EVENTS (3)
                </div>
                <Link href="#" className="text-xs text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-300 transition-colors duration-200">
                View more
              </Link>
              </div>
              {displayEvents.map((event, index) => {
                const getStatusConfig = (status: string) => {
                  switch (status) {
                    case 'pending':
                      return {
                        icon: 'schedule',
                        color: 'text-yellow-600 dark:text-yellow-400',
                        bgColor: 'bg-yellow-50 dark:bg-yellow-900/20',
                        labelColor: 'yellow',
                        borderColor: 'border-yellow-200 dark:border-yellow-800',
                        dotColor: 'bg-yellow-500 animate-pulse',
                        label: 'Pending',
                        shortLabel: 'P',
                        description: 'Processing...'
                      };
                    case 'discarded':
                      return {
                        icon: 'block',
                        color: 'text-zinc-600 dark:text-zinc-400',
                        bgColor: 'bg-zinc-50 dark:bg-zinc-900/20',
                        labelColor: 'zinc',
                        borderColor: 'border-zinc-200 dark:border-zinc-800',
                        dotColor: 'bg-zinc-500',
                        label: 'Discarded',
                        shortLabel: 'D',
                        description: 'Filtered out'
                      };
                    case 'forwarded':
                      return {
                        icon: 'check_circle',
                        color: 'text-green-600 dark:text-green-400',
                        bgColor: 'bg-green-50 dark:bg-green-900/20',
                        labelColor: 'green',
                        borderColor: 'border-green-200 dark:border-green-800',
                        dotColor: 'bg-green-500',
                        label: 'Forwarded',
                        shortLabel: 'C',
                        description: event.processingTime ? `${event.processingTime}ms` : 'Completed'
                      };
                    default:
                      return {
                        icon: 'bolt',
                        color: 'text-zinc-600 dark:text-zinc-400',
                        bgColor: 'bg-zinc-50 dark:bg-zinc-800',
                        labelColor: 'zinc',
                        borderColor: 'border-zinc-200 dark:border-zinc-700',
                        dotColor: 'bg-zinc-500',
                        label: 'Unknown',
                        shortLabel: '?',
                        description: ''
                      };
                  }
                };

                const statusConfig = getStatusConfig(event.status);
                const timeAgo = compactEvent ? `${index * 1}m` : `${index * 1}m ago`;

                if (compactEvent) {
                  // Compact mode - single line with minimal info
                  return (
                    
                      <div className="flex items-center gap-2 p-2 bg-gray-50 dark:bg-zinc-800 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors duration-150 cursor-pointer">
                        {/* Compact Status Indicator */}
                        <div className="flex items-center gap-2">
                          <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full flex-shrink-0`}></div>
                          <span className={`text-xs font-medium ${statusConfig.color}`}>
                            {statusConfig.label}
                          </span>
                        </div>
                        
                        {/* Event URL */}
                        <span className={event.status === 'discarded' ? 'text-sm font-mono text-gray-800 dark:text-zinc-200 truncate flex-1 line-through opacity-60' : "text-sm font-mono text-gray-800 dark:text-zinc-200 truncate flex-1"}>
                          {truncateUrl(event.url, 22)}
                        </span>
                        
                        <Tippy
                          key={event.id}
                          content={
                            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-3 min-w-[260px]">
                              {event.timestamp}
                            </div>
                          }
                          placement="top"
                          trigger="mouseenter"
                          delay={[200, 100]}
                          className="z-50"
                        >    
                        {/* Time */}
                          <span className="text-xs text-zinc-500 dark:text-zinc-400 flex-shrink-0 w-6 text-right">
                            {timeAgo}
                          </span>
                        </Tippy>
                      </div>
                    
                  );
                } else {
                  // Expanded mode - multi-line with detailed status info
                  return (
                    <Tippy
                      key={event.id}
                      content={
                        <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[280px]">
                          <div className="flex items-start gap-3 mb-3">
                            <div className="flex-1">
                              <div className="text-sm font-semibold text-zinc-900 dark:text-white">
                                Event {statusConfig.label}
                              </div>
                              <div className="text-xs text-zinc-500 dark:text-zinc-400">
                                {timeAgo}
                              </div>
                            </div>
                            <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full mt-2`}></div>
                          </div>
                          
                          <div className="space-y-2 mb-3">
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-zinc-500 dark:text-zinc-400">URL:</span>
                              <span className="font-mono text-zinc-700 dark:text-zinc-300 truncate max-w-48" title={event.url}>
                                {event.url}
                              </span>
                            </div>
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-zinc-500 dark:text-zinc-400">Status:</span>
                              <span className={`${statusConfig.color} font-medium`}>
                                {statusConfig.label} - {statusConfig.description}
                              </span>
                            </div>
                            {event.type && (
                              <div className="flex items-center justify-between text-xs">
                                <span className="text-zinc-500 dark:text-zinc-400">Type:</span>
                                <span className="text-zinc-700 dark:text-zinc-300 font-mono">
                                  {event.type}
                                </span>
                              </div>
                            )}
                          </div>
                          
                          <div className="pt-2 border-t border-zinc-200 dark:border-zinc-700 text-xs text-zinc-500 dark:text-zinc-400">
                            {event.status === 'pending' && 'Event is being processed and will trigger workflows when complete'}
                            {event.status === 'discarded' && 'Event was filtered out and did not trigger any workflows'}
                            {event.status === 'forwarded' && 'Event successfully triggered workflow execution'}
                          </div>
                        </div>
                      }
                      interactive={true}
                      placement="bottom"
                      trigger="mouseenter"
                      delay={[200, 100]}
                      className="z-50"
                    >
                      <div className={`border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-md hover:shadow-sm transition-all duration-200 cursor-pointer`}>
                        {/* Main event row */}
                        <div className="flex items-center justify-between px-3 pt-2">
                          <div className="flex items-center gap-2">
                            <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full flex-shrink-0`}></div>
                            <span className={`text-xs font-medium ${statusConfig.color}`}>
                              {statusConfig.label}
                            </span>
                            <span className="text-xs text-zinc-500 dark:text-zinc-400">
                              • {statusConfig.description}
                            </span>
                           
                          </div>
                          <span className="text-xs text-zinc-500 dark:text-zinc-400 flex-shrink-0">
                            {timeAgo}
                          </span>
                        </div>
                        <div className="flex items-center gap-3 p-3">
                          
                          
                          {/* Event URL */}
                          <div className="flex-1 min-w-0">
                            <span className={event.status === 'discarded' ? 'text-sm font-mono text-gray-800 dark:text-zinc-200 truncate block line-through opacity-60' : "text-sm font-mono text-gray-800 dark:text-zinc-200 truncate block"}>
                              {truncateUrl(event.url, 35)}
                            </span>
                          </div>
                          
                          {/* Time */}
                          
                        </div>
                        
                        {/* Status information row */}
                        
                      </div>
                    </Tippy>
                  );
                }
              })}
              
            </div>
          ) : (
            /* noEvents variant - empty state */
            <div className="bg-zinc-50 dark:bg-zinc-800 px-4 rounded-lg">
              
              
              {/* Empty State */}
              <EmptyState
                size="xs"
                icon="sensors"
                animated={true}
                animationType="pulse"
                title='Ready to receive events'
                body={`Listening to changes in your Semaphore project`}
                className="pt-6 pb-4"
              />
            </div>
          )}
        </div>

        {/* React Flow Handles */}
        <Handle
          type="target"
          position={Position.Left}
          className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md !hidden"
          aria-label="Input connection point"
        />
        <Handle
          type="source"
          position={Position.Right}
          className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md"
          aria-label="Output connection point"
        />
        </div>
      </>
    );
  }

  // Edit Mode
  return (
    <>
      <div 
      className={clsx(
        'bg-white dark:bg-zinc-800 relative transition-all duration-200 hover:shadow-lg min-w-[320px]',
        
      )}
      style={{ width: 380, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
      role="article"
    >
      {/* Edit Header */}
      <div 
          className="action-buttons absolute -top-14 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-xs rounded-lg p-1 border border-gray-200 dark:border-zinc-600 z-50"
          onClick={(e) => e.stopPropagation()}
        >
         

            <Button
              type="button"
              plain
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="code" size="md"/>
              Code
            </Button>
            
          
            
            
            <Dropdown>
              <DropdownButton plain className='flex items-center gap-2 !pr-1'>
                <MaterialSymbol name="save" size="md"/> 
                Save
                <MaterialSymbol name="expand_more" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2' onClick={handleSaveAndCommit}><DropdownLabel>Save & Commit</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2' onClick={handleSave}><DropdownLabel>Save as Draft</DropdownLabel></DropdownItem>
               
              </DropdownMenu>
            </Dropdown>
           
            <Button
              type="button"
              plain
              onClick={handleCancel}
              className="flex items-center gap-2"
            >
              
              Cancel
            </Button>
            <Tippy content="More options" placement="top">
            <Dropdown>
              <DropdownButton plain>
                <MaterialSymbol name="more_vert" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="play_arrow" size="md"/><DropdownLabel>Run</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="tune" size="md"/><DropdownLabel>Advanced configuration</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="menu_book" size="md"/><DropdownLabel>Documentation</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2 text-red-600 dark:text-red-200' color='red'><MaterialSymbol name="delete" size="md"/><DropdownLabel>Delete</DropdownLabel></DropdownItem>

              </DropdownMenu>
            </Dropdown>
          </Tippy>
         
          
          
        </div>

      {/* Edit Form */}
      <div>
        {/* Main content area with blue border */}
        <div className="border-2 border-blue-500 rounded-lg bg-white dark:bg-zinc-800">
          {/* Header with icon and title */}
          <div className='flex flex-col p-4'>
            <div className="flex items-center gap-2 mb-2">
              {data.eventSourceType === 'webhook' ? (
                <button
                  onClick={() => setShowIconModal(true)}
                  className="flex items-center content-center bg-zinc-100 dark:bg-zinc-700 rounded-md w-10 h-10 hover:bg-zinc-200 dark:hover:bg-zinc-600 transition-colors cursor-pointer border border-gray-400 dark:border-zinc-600 border-dashed"
                  title="Click to change icon"
                >
                  <div className="m-auto relative">
                    {renderIcon(currentIcon)}
                    {currentIcon != 'webhook' && ( <div className='absolute bottom-[-12px] right-[-12px] bg-white dark:bg-zinc-700 rounded-full w-5 h-5 flex items-center justify-center border border-gray-400 dark:border-zinc-600'><MaterialSymbol name="webhook" size="sm" className="" /></div>)}
                  </div>
                </button>
              ) : (
                <div className="flex items-center content-center bg-zinc-100 dark:bg-zinc-700 rounded-md w-10 h-10">
                  {data.eventSourceType === 'http' ? (
                    <MaterialSymbol name="rocket_launch" size="lg" className="text-gray-700 dark:text-zinc-300 m-auto" />
                  ) : data.icon === 'semaphore' ? (
                    <img width={24} height={24} className='m-auto' src='/images/semaphore-logo-sign-black.svg' alt="Semaphore" />
                  ) : data.icon === 'github' ? (
                    <img width={24} height={24} className='m-auto' src='/images/github-logo.svg' alt="GitHub" />
                  ) : (
                    <img width="24" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt=""/>
                  )}
                </div>
              )}
              
                <h2 className="text-md font-semibold text-gray-900 dark:text-white">
                  {data.title}
                </h2>
              
            
            </div>
            <span className='text-xs font-medium text-zinc-600 dark:text-zinc-400'>Description</span>
          </div>
          

          {/* Semaphore integration section */}
          <div className="space-y-3 border-t border-gray-200 dark:border-zinc-600 p-4">
            {inlineIntegration ? (
              /* Inline integration form */
              <div className="space-y-4 border border-gray-300 dark:border-zinc-600 rounded-lg p-4">
                <Field>
                  <Label className="text-sm font-medium text-gray-900 dark:text-white">
                    {data.icon === 'github' ? 'GitHub organization/owner URL' : 'Semaphore Organization URL'}
                  </Label>
                  <Input
                    type="url"
                    value={inlineIntegrationData.orgUrl}
                    onChange={(e) => setInlineIntegrationData(prev => ({ ...prev, orgUrl: e.target.value }))}
                    placeholder={data.icon === 'github' ? 'https://github.com/owner' : 'https://your-org.semaphoreci.com'}
                    className="w-full"
                  />
                </Field>
                
                <Field>
                  <Label className="text-sm font-medium text-gray-900 dark:text-white">
                    API Token
                  </Label>
                  <Input
                    type="password"
                    value={inlineIntegrationData.apiToken}
                    onChange={(e) => setInlineIntegrationData(prev => ({ ...prev, apiToken: e.target.value }))}
                    placeholder="Enter your API token"
                    className="w-full"
                  />
                </Field>
              </div>
            ) : availableIntegrations.length > 0 ? (
              /* Show integration dropdown */
              <div className="space-y-3">
                <Dropdown>
                  <DropdownButton outline className="flex items-center w-full !justify-between">
                    {selectedIntegration 
                      ? availableIntegrations.find(i => i.id === selectedIntegration)?.name || 'Select integration'
                      : (data.icon === 'github' ? 'Select GitHub integration' : 'Select Semaphore integration')
                    }
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu>
                    {availableIntegrations.map((integration) => (
                      <DropdownItem
                        key={integration.id}
                        onClick={() => setSelectedIntegration(integration.id)}
                      >
                        <DropdownLabel>{integration.name}</DropdownLabel>
                      </DropdownItem>
                    ))}
                    <DropdownItem onClick={handleCreateIntegration}>
                      <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400">
                        <MaterialSymbol name="add" size="sm" />
                        <span>Create new integration</span>
                      </div>
                    </DropdownItem>
                  </DropdownMenu>
                </Dropdown>
                <Field>
                  <Label className="text-sm font-medium text-gray-900 dark:text-white">{data.icon === 'github' ? 'Repository Name' : 'Semaphore Project'}</Label>
                  <Input  type="text" placeholder={data.icon === 'github' ? 'owner/repo' : 'Enter your Semaphore project name'} className="w-full" />
                </Field>
              </div>
            ) : (
              /* Empty state */
              <div className="text-center py-8 bg-zinc-50 dark:bg-zinc-700 border border-gray-200 dark:border-gray-700 rounded-md">
                <div className="text-gray-500 dark:text-zinc-400 mb-3">
                  {data.icon === 'github' 
                    ? "Looks like you haven't connected any GitHub accounts yet"
                    : "Looks like you haven't connected any Semaphore organizations yet"}
                </div>
                <Button
                  onClick={handleCreateIntegration}
                  color='blue'
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md text-sm font-medium"
                >
                  Create integration
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* React Flow Handles */}
      <Handle 
        type="target" 
        position={Position.Left} 
        className="w-3 h-3 bg-gray-400 border-2 border-white" 
      />
      <Handle 
        type="source" 
        position={Position.Right} 
        className="w-3 h-3 bg-blue-600 border-2 border-white" 
      />

      {/* Integration Creation Modal */}
      <Dialog 
        open={showIntegrationModal} 
        onClose={() => setShowIntegrationModal(false)}
        className="relative z-50"
        size="md"
      >
        <DialogTitle>Create {data.icon === 'github' ? 'GitHub' : 'Semaphore'} Integration</DialogTitle>
        <DialogDescription>
          New integration will be saved to integrations page. Manage integrations  
          <Link href="/integrations" className='text-blue-600 dark:text-blue-400'> here</Link>.
        </DialogDescription>
        
        <DialogBody className="space-y-6">
          {/* Org/Repo URL */}
          <Field>
            <Label className="text-sm font-medium text-gray-900 dark:text-white">
              {data.icon === 'github' ? 'GitHub organization/owner URL' : 'Semaphore Org URL'}
            </Label>
            <Input
              type="url"
              value={integrationData.orgUrl}
              onChange={(e) => setIntegrationData(prev => ({ ...prev, orgUrl: e.target.value }))}
              placeholder={data.icon === 'github' ? 'https://github.com/owner' : 'https://your-org.semaphoreci.com'}
              className="w-full"
            />
          </Field>

          {/* Integration Name */}
          <Field>
            <Label className="text-sm font-medium text-gray-900 dark:text-white">
              Integration Name
            </Label>
            <Input
              type="text"
              value={integrationData.name}
              onChange={(e) => setIntegrationData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Enter integration name"
              className="w-full"
            />
          </Field>

          {data.icon === 'github' && (
            <div className="rounded-md border border-gray-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 p-4">
              <div className="flex items-start gap-3">
                <div className="mt-0.5 text-zinc-600 dark:text-zinc-300">
                  <MaterialSymbol name="info" size="md" />
                </div>
                <div className="flex-1">
                  <div className="text-sm font-medium text-gray-900 dark:text-white">GitHub Personal Access Token (PAT) required</div>
                  <p className="text-sm text-zinc-700 dark:text-zinc-300 mt-1">
                    To connect GitHub, create a fine‑grained Personal Access Token and provide it as the API token.
                  </p>
                  <button
                    type="button"
                    className="mt-2 text-sm text-blue-600 dark:text-blue-300 hover:underline"
                    aria-expanded={showGitHubPatInfo}
                    onClick={() => setShowGitHubPatInfo(v => !v)}
                  >
                    {showGitHubPatInfo ? 'Hide details' : 'Show how to configure PAT'}
                  </button>
                  {showGitHubPatInfo && (
                    <div className="mt-3 space-y-2 text-sm text-zinc-700 dark:text-zinc-300">
                      <p>When creating a fine‑grained PAT</p>
                      <div><strong>Chose the access scope:</strong></div>
                      <ul className="list-disc ml-5 mt-1 space-y-1">
                        <li>All repositories</li>
                        <li>Or select specific repositories</li>
                      </ul>
                      <div className="mt-2"><strong>Set required permissions:</strong></div>
                      <ul className="list-disc ml-5 mt-1 space-y-1">
                        <li>Actions - Read AND Write</li>
                        <li>Webhooks - Read AND Write</li>
                      </ul>
                      <p className="text-xs text-zinc-600 dark:text-zinc-400">
                        Tip: You can manage or rotate the PAT anytime in your GitHub developer settings.
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* API Token Section */}
          <div className="space-y-4">
            <div className="text-sm font-medium text-gray-900 dark:text-white flex items-center justify-between">
              API Token
              <Link href={'#'} className='flex items-center gap-2 text-blue-600 dark:text-blue-200 hidden'>
                <MaterialSymbol name="add" size="md"/>Add new secret
              </Link>
            </div>
            
            {/* API Token Tabs */}
            <div>
              <ControlledTabs
                tabs={apiTokenTabs}
                activeTab={apiTokenTab}
                variant='pills'
                className='w-full'
                onTabChange={(tabId) => setApiTokenTab(tabId as 'existing' | 'new')}
              />
              
              <div className="pt-4">
                {apiTokenTab === 'existing' ? (
                  /* Select existing secret */
                  <div className="space-y-4">
                    <Field>
                      
                      <Dropdown>
                        <DropdownButton outline className='flex items-center w-full !justify-between'>
                          {integrationData.apiToken.secretName || 'Select secret'}
                          <MaterialSymbol name="keyboard_arrow_down" />
                        </DropdownButton>
                        <DropdownMenu anchor="bottom start">
                          {secrets.map((secret) => (
                            <DropdownItem
                              key={secret.id}
                              onClick={() => setIntegrationData(prev => ({
                                ...prev,
                                apiToken: { ...prev.apiToken, secretName: secret.name, secretKey: secret.keys[0]?.key || '' }
                              }))}
                            >
                              <DropdownLabel>{secret.name}</DropdownLabel>
                            </DropdownItem>
                          ))}
                        </DropdownMenu>
                      </Dropdown>
                    </Field>
                    {selectedExistingSecret && (
                      <Field className='flex items-start gap-3 w-full'>
                        <div className='w-50'>
                          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                            Key name
                          </Label>
                          <Input
                            type="text"
                            value={selectedExistingSecret.keys[0]?.key || ''}
                            readOnly
                            className="w-full bg-gray-50 dark:bg-zinc-800 cursor-default"
                          />
                        </div>
                        <div className='w-50'>
                          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                            Value
                          </Label>
                          <Input
                            type="password"
                            value={selectedExistingSecret.keys[0]?.value || ''}
                            readOnly
                            disabled
                            className="w-full bg-gray-50 dark:bg-zinc-800 cursor-not-allowed opacity-75"
                          />
                        </div>
                      </Field>
                    )}
                  </div>
                ) : (
                  /* Create new secret */
                  <div className="space-y-4 w-full">
                    <Text className='text-xs text-gray-500 dark:text-zinc-400'>
                      New secret will be created in your canvas secrets. 
                      You can review and manage your secrets in the secrets tab <Link href="#" className='text-blue-600 dark:text-blue-200'>here</Link>
                    </Text>

                    <Field className='flex items-start gap-3 w-full'>
                      <div className='w-50'>
                        <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                          Key name
                        </Label>
                        <Input
                          type="text"
                          value="my-api-token"
                          className="w-full"
                        />
                      </div>
                      <div className='w-50'>
                        <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                          Value
                        </Label>
                        <Input
                          type="password"
                          value={newSecretToken}
                          onChange={(e) => setNewSecretToken(e.target.value)}
                          placeholder="Enter your API token"
                          className="w-full"
                        />
                      </div>
                    </Field>
                  </div>
                )}
              </div>
            </div>
          </div>
        </DialogBody>

        <DialogActions>
          <Button
            onClick={() => setShowIntegrationModal(false)}
          >
            Cancel
          </Button>
          <Button
            color='blue'
            onClick={handleSaveIntegration}
            disabled={!integrationData.name || !integrationData.orgUrl || !integrationData.apiToken.secretName || !integrationData.apiToken.secretKey}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Icon Selection Modal */}
      <Dialog
        open={showIconModal}
        onClose={handleIconModalClose}
        className="relative z-50"
        size="md"
      >
        <DialogTitle>Choose Icon</DialogTitle>
        <DialogDescription>
          Select an icon to represent your webhook event source
        </DialogDescription>
        
        <DialogBody className="space-y-4">
          {/* Search Input */}
          <div>
            <Field>
              
              <Input
                type="text"
                value={iconSearchQuery}
                onChange={(e) => setIconSearchQuery(e.target.value)}
                placeholder="Search by name or type (e.g., webhook, docker, github...)"
                className="w-full"
                autoFocus
              />
            </Field>
          </div>

          {/* Material Symbol Icons */}
          {filteredMaterialIcons.length > 0 && (
            <div>
              <div className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
                Icons ({filteredMaterialIcons.length})
              </div>
              <div className="grid grid-cols-6 gap-3">
                {filteredMaterialIcons.map((iconOption) => (
                  <button
                    key={iconOption.name}
                    onClick={() => handleIconChange(iconOption.name)}
                    className={`p-3 rounded-lg border transition-all duration-200 hover:shadow-md ${
                      currentIcon === iconOption.name
                        ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 shadow-md'
                        : 'border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-800 text-gray-600 dark:text-zinc-400 hover:border-gray-300 dark:hover:border-zinc-600 hover:bg-gray-100 dark:hover:bg-zinc-700'
                    }`}
                    title={iconOption.label}
                  >
                    <MaterialSymbol name={iconOption.name} size="lg" />
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* DevOps Tool Icons */}
          {filteredDevopsIcons.length > 0 && (
            <div>
              <div className="text-sm font-semibold text-gray-900 dark:text-white mb-3">
                DevOps Tools ({filteredDevopsIcons.length})
              </div>
              <div className="grid grid-cols-6 gap-3">
                {filteredDevopsIcons.map((iconOption) => (
                  <button
                    key={iconOption.name}
                    onClick={() => handleIconChange(iconOption.name)}
                    className={`p-3 rounded-lg border transition-all duration-200 hover:shadow-md ${
                      currentIcon === iconOption.name
                        ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 shadow-md'
                        : 'border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-800 hover:border-gray-300 dark:hover:border-zinc-600 hover:bg-gray-100 dark:hover:bg-zinc-700'
                    }`}
                    title={iconOption.label}
                  >
                    <img 
                      width={24} 
                      height={24} 
                      src={iconOption.icon} 
                      alt={iconOption.label} 
                      className="flex-shrink-0"
                    />
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* No Results */}
          {iconSearchQuery && filteredMaterialIcons.length === 0 && filteredDevopsIcons.length === 0 && (
            <div className="text-center py-8 text-gray-500 dark:text-zinc-400">
              <MaterialSymbol name="search_off" size="lg" className="mb-2" />
              <div className="text-sm">No icons found for "{iconSearchQuery}"</div>
              <div className="text-xs mt-1">Try searching for a different term</div>
            </div>
          )}
        </DialogBody>

        <DialogActions>
          <Button
            color='white'
            onClick={handleIconModalClose}
          >
            Cancel
          </Button>
        </DialogActions>
      </Dialog>

      </div>

      
    </>
  );
}