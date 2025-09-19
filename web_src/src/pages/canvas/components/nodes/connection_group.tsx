import { useState, useMemo, useCallback } from 'react';
import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { ConnectionGroupNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { useCreateConnectionGroup, useUpdateConnectionGroup, useDeleteConnectionGroup } from '@/hooks/useCanvasData';
import { SuperplaneConnection, GroupByField, SpecTimeoutBehavior, SuperplaneConnectionGroup } from '@/api-client';
import { ConnectionGroupEditModeContent } from '../ConnectionGroupEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { EditModeActionButtons } from '../EditModeActionButtons';
import { twMerge } from 'tailwind-merge';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { createConnectionGroupDuplicate, focusAndEditNode } from '../../utils/nodeDuplicationUtils';
import { EmitEventModal } from '../EmitEventModal';

export default function ConnectionGroupNode(props: NodeProps<ConnectionGroupNodeType>) {
  const isNewNode = props.id && /^\d+$/.test(props.id);
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ name: string; description?: string; connections: SuperplaneConnection[]; groupByFields: GroupByField[]; timeout?: number; timeoutBehavior?: SpecTimeoutBehavior; isValid: boolean } | null>(null);
  const [connectionGroupName, setConnectionGroupName] = useState(props.data.name || '');
  const [connectionGroupDescription, setConnectionGroupDescription] = useState(props.data.description || '');
  const [nameError, setNameError] = useState<string | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);
  const [showEmitEventModal, setShowEmitEventModal] = useState(false);
  const { updateConnectionGroup, setEditingConnectionGroup, removeConnectionGroup, addConnectionGroup, setFocusedNodeId } = useCanvasStore();
  const allConnectionGroups = useCanvasStore(state => state.connectionGroups);

  const currentConnectionGroup = useCanvasStore(state =>
    state.connectionGroups.find(cg => cg.metadata?.id === props.id)
  );
  const nodes = useCanvasStore(state => state.nodes);

  const isPartiallyBroken = useMemo(() => {
    if (!currentConnectionGroup || isNewNode)
      return false;

    const hasNoConnections = currentConnectionGroup.spec?.connections?.length === 0

    const hasInvalidConnections = currentConnectionGroup.spec?.connections?.some(connection => {
      return !nodes.some(node => node?.data?.name === connection.name)
    })

    return hasNoConnections || hasInvalidConnections
  }, [currentConnectionGroup, isNewNode, nodes])

  const validateConnectionGroupName = (name: string) => {
    if (!name || name.trim() === '') {
      setNameError('Connection group name is required');
      return false;
    }

    const isDuplicate = allConnectionGroups.some(cg =>
      cg.metadata?.name?.toLowerCase() === name.toLowerCase() &&
      cg.metadata?.id !== props.id
    );

    if (isDuplicate) {
      setNameError('A connection group with this name already exists');
      return false;
    }

    setNameError(null);
    return true;
  };

  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const createConnectionGroupMutation = useCreateConnectionGroup(canvasId);
  const updateConnectionGroupMutation = useUpdateConnectionGroup(canvasId);
  const deleteConnectionGroupMutation = useDeleteConnectionGroup(canvasId);
  const focusedNodeId = useCanvasStore(state => state.focusedNodeId);
  const groupByFields = props.data.groupBy?.fields || [];

  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingConnectionGroup(props.id);

    setConnectionGroupName(props.data.name);
    setConnectionGroupDescription(props.data.description || '');
  };

  const handleSaveConnectionGroup = async () => {
    if (!currentFormData || !currentConnectionGroup) {
      return;
    }

    if (!validateConnectionGroupName(connectionGroupName)) {
      return;
    }

    const isTemporaryId = currentConnectionGroup.metadata?.id && /^\d+$/.test(currentConnectionGroup.metadata.id);
    const isNewConnectionGroup = !currentConnectionGroup.metadata?.id || isTemporaryId;

    try {
      setApiError(null);

      if (isNewConnectionGroup) {

        const result = await createConnectionGroupMutation.mutateAsync({
          name: connectionGroupName,
          description: connectionGroupDescription,
          connections: currentFormData.connections,
          groupByFields: currentFormData.groupByFields,
          timeout: currentFormData.timeout,
          timeoutBehavior: currentFormData.timeoutBehavior
        });

        const newConnectionGroup = result.data?.connectionGroup;
        if (newConnectionGroup) {
          removeConnectionGroup(props.id);
        }
      } else if (!isNewConnectionGroup) {

        if (!currentConnectionGroup.metadata?.id) {
          throw new Error('Connection Group ID is required for update');
        }

        await updateConnectionGroupMutation.mutateAsync({
          connectionGroupId: currentConnectionGroup.metadata.id,
          name: connectionGroupName,
          description: connectionGroupDescription,
          connections: currentFormData.connections,
          groupByFields: currentFormData.groupByFields,
          timeout: currentFormData.timeout,
          timeoutBehavior: currentFormData.timeoutBehavior
        });

        updateConnectionGroup({
          ...currentConnectionGroup,
          metadata: {
            ...currentConnectionGroup.metadata,
            name: connectionGroupName,
            description: connectionGroupDescription
          },
          spec: {
            ...currentConnectionGroup.spec!,
            connections: currentFormData.connections,
            groupBy: {
              fields: currentFormData.groupByFields
            },
            timeout: currentFormData.timeout,
            timeoutBehavior: currentFormData.timeoutBehavior
          }
        });

        props.data.name = connectionGroupName;
        props.data.description = connectionGroupDescription;
      }
      setIsEditMode(false);
      setEditingConnectionGroup(null);
      setCurrentFormData(null);
    } catch (error) {
      setApiError(((error as Error)?.message) || error?.toString() || 'An error occurred');
    }
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingConnectionGroup(null);
    setCurrentFormData(null);

    setConnectionGroupName(props.data.name);
    setConnectionGroupDescription(props.data.description || '');
  };

  const handleDiscardConnectionGroup = async () => {
    if (currentConnectionGroup?.metadata?.id) {
      const isTemporaryId = /^\d+$/.test(currentConnectionGroup.metadata.id);
      const isRealConnectionGroup = !isTemporaryId;

      if (isRealConnectionGroup) {
        try {
          await deleteConnectionGroupMutation.mutateAsync(currentConnectionGroup.metadata.id);
        } catch (error) {
          console.error('Failed to delete connection group:', error);
          return;
        }
      }
      removeConnectionGroup(currentConnectionGroup.metadata.id);
    }
    setShowDiscardConfirm(false);
  };

  const handleConnectionGroupNameChange = (newName: string) => {
    setConnectionGroupName(newName);
    validateConnectionGroupName(newName);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: newName
      });
    }
  };

  const handleConnectionGroupDescriptionChange = (newDescription: string) => {
    setConnectionGroupDescription(newDescription);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        description: newDescription
      });
    }
  };

  const handleDuplicateConnectionGroup = () => {
    if (!currentConnectionGroup) return;

    const duplicatedConnectionGroup = createConnectionGroupDuplicate(currentConnectionGroup, allConnectionGroups);
    addConnectionGroup(duplicatedConnectionGroup, true);

    focusAndEditNode(
      duplicatedConnectionGroup.metadata?.id || '',
      setFocusedNodeId,
      setEditingConnectionGroup
    );
  };

  const handleYamlApply = useCallback((updatedData: unknown) => {
    // Handle YAML data application for connection group
    const yamlData = updatedData as SuperplaneConnectionGroup;

    if (yamlData.metadata?.name) {
      setConnectionGroupName(yamlData.metadata.name);
    }
    if (yamlData.metadata?.description) {
      setConnectionGroupDescription(yamlData.metadata.description);
    }

    // Update form data if available
    if (yamlData.spec && currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: yamlData.metadata?.name || connectionGroupName,
        description: yamlData.metadata?.description || connectionGroupDescription,
        connections: yamlData.spec.connections || currentFormData.connections,
        groupByFields: yamlData.spec.groupBy?.fields || currentFormData.groupByFields,
        timeout: yamlData.spec.timeout || currentFormData.timeout,
        timeoutBehavior: yamlData.spec.timeoutBehavior || currentFormData.timeoutBehavior,
        isValid: currentFormData.isValid // Keep current validation state
      });
    }
  }, [currentFormData, connectionGroupName, connectionGroupDescription]);

  const handleDataChange = useCallback((data: { name: string; description?: string; connections: SuperplaneConnection[]; groupByFields: GroupByField[]; timeout?: number; timeoutBehavior?: SpecTimeoutBehavior; isValid: boolean }) => {
    setCurrentFormData(data);
    setApiError(null);
  }, []);

  const borderColor = useMemo(() => {
    if (isPartiallyBroken) {
      return 'border-red-400 dark:border-red-200'
    }

    if (props.selected || focusedNodeId === props.id) {
      return 'border-blue-400'
    }
    return 'border-gray-200 dark:border-gray-700'
  }, [props.selected, focusedNodeId, props.id, isPartiallyBroken])

  return (
    <div
      className={twMerge(`bg-white dark:bg-zinc-800 rounded-lg shadow-lg border-2 relative`, borderColor)}
      style={{ width: '390px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {(focusedNodeId === props.id || isEditMode) && (
        <EditModeActionButtons
          isNewNode={!!isNewNode}
          onSave={handleSaveConnectionGroup}
          onCancel={handleCancelEdit}
          onDiscard={() => setShowDiscardConfirm(true)}
          onEdit={handleEditClick}
          onDuplicate={!isNewNode ? handleDuplicateConnectionGroup : undefined}
          isEditMode={isEditMode}
          entityType="connection group"
          entityData={currentFormData ? {
            metadata: {
              name: connectionGroupName,
              description: connectionGroupDescription
            },
            spec: {
              connections: currentFormData.connections,
              groupBy: {
                fields: currentFormData.groupByFields
              },
              timeout: currentFormData.timeout,
              timeoutBehavior: currentFormData.timeoutBehavior
            }
          } : (currentConnectionGroup ? {
            metadata: {
              name: currentConnectionGroup.metadata?.name,
              description: currentConnectionGroup.metadata?.description
            },
            spec: {
              connections: currentConnectionGroup.spec?.connections || [],
              groupBy: {
                fields: currentConnectionGroup.spec?.groupBy?.fields || []
              },
              timeout: currentConnectionGroup.spec?.timeout,
              timeoutBehavior: currentConnectionGroup.spec?.timeoutBehavior || 'TIMEOUT_BEHAVIOR_DROP'
            }
          } : null)}
          onYamlApply={handleYamlApply}
        />
      )}

      {/* Header Section */}
      <div className="mt-1 px-4 py-4 justify-between items-start border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-start justify-between w-full">
        <div className="flex items-start flex-1 min-w-0">
          <div className='max-w-8 mt-1 flex items-center justify-center'>
            <MaterialSymbol name="account_tree" size="lg" />
          </div>
          <div className="flex-1 min-w-0 ml-2">
            <div className="mb-1">
              <InlineEditable
                value={connectionGroupName}
                onSave={handleConnectionGroupNameChange}
                placeholder="Connection group name"
                className={twMerge(`font-bold text-gray-900 dark:text-gray-100 text-base text-left px-2 py-1`,
                  nameError && isEditMode ? 'border border-red-500 rounded-lg' : '',
                  isEditMode ? 'text-sm' : '')}
                isEditMode={isEditMode}
                autoFocus={!!isNewNode}
                dataTestId="event-source-name-input"
              />
              {nameError && isEditMode && (
                <div className="text-xs text-red-600 text-left mt-1 px-2">
                  {nameError}
                </div>
              )}
            </div>
            <div>
              {isEditMode && <InlineEditable
                value={connectionGroupDescription}
                onSave={handleConnectionGroupDescriptionChange}
                placeholder={isEditMode ? "Add description..." : ""}
                className="text-gray-600 dark:text-gray-400 text-sm text-left px-2 py-1"
                isEditMode={isEditMode}
              />}
            </div>
          </div>
        </div>
        {!isEditMode && currentConnectionGroup?.metadata?.id && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              setShowEmitEventModal(true);
            }}
            className="ml-2 p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:text-blue-400 dark:hover:bg-blue-900/20 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1"
            title="Emit a test event"
          >
            <MaterialSymbol name="send" size="sm" />
          </button>
        )}
        </div>
        {!isEditMode && (
          <div className="text-xs text-left text-gray-600 dark:text-gray-400 w-full mt-1">{connectionGroupDescription || ''}</div>
        )}
      </div>

      {isEditMode ? (
        <ConnectionGroupEditModeContent
          data={{
            ...props.data,
            name: connectionGroupName,
            description: connectionGroupDescription,
            ...(currentFormData && {
              connections: currentFormData.connections,
              groupBy: { fields: currentFormData.groupByFields },
              timeout: currentFormData.timeout,
              timeoutBehavior: currentFormData.timeoutBehavior
            })
          }}
          currentConnectionGroupId={props.id}
          apiError={apiError}
          onDataChange={handleDataChange}
        />
      ) : (
        <>
          {/* Group By Section */}
          <div className="px-3 py-3">
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wide">Group By Fields</div>
              <div className="text-xs text-gray-600 dark:text-gray-400">
                {groupByFields.length} fields
              </div>
            </div>

            <div className="space-y-1">
              {groupByFields.length > 0 ? (
                groupByFields.map((field, index) => (
                  <div key={index} className="bg-gray-100 dark:bg-gray-700 rounded p-2">
                    <div className="flex justify-start items-center gap-3 overflow-hidden">
                      <span className="text-sm text-gray-600 dark:text-gray-400">
                        <i className="material-icons f3 fill-black rounded-full bg-[var(--washed-blue)] black-60 p-1">label</i>
                      </span>
                      <span className="truncate font-medium">{field.name}</span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">→</span>
                      <span className="truncate text-sm">{field.expression}</span>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-500 dark:text-gray-400 italic py-2">No group by fields configured</div>
              )}
            </div>
          </div>

          {/* Connections Section */}
          <div className="px-3 py-3 border-t w-full border-gray-200 dark:border-zinc-700">
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wide">Connections</div>
              <div className="text-xs text-gray-600 dark:text-gray-400">
                {props.data.connections?.length || 0} connections
              </div>
            </div>

            <div className="space-y-1">
              {props.data.connections?.length ? (
                props.data.connections.map((connection, index) => (
                  <div key={index} className="bg-gray-100 dark:bg-gray-700 rounded p-2">
                    <div className="flex justify-between items-center gap-3 overflow-hidden">
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          <i className="material-icons f3 fill-black rounded-full bg-[var(--washed-green)] black-60 p-1">link</i>
                        </span>
                        <span className="truncate font-medium">{connection.name}</span>
                      </div>
                      <span className="text-xs bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-400 px-2 py-0.5 rounded">
                        {connection.type?.toString().replace('TYPE_', '').replace('_', ' ').toLowerCase()}
                      </span>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-500 dark:text-gray-400 italic py-2">No connections configured</div>
              )}
            </div>
          </div>
        </>
      )}

      <CustomBarHandle type="target" connections={props.data.connections} />
      <CustomBarHandle type="source" />

      {/* Discard Confirmation Dialog */}
      <ConfirmDialog
        isOpen={showDiscardConfirm}
        title="Discard Connection Group"
        message="Are you sure you want to discard this connection group? This action cannot be undone."
        confirmText="Discard"
        cancelText="Cancel"
        confirmVariant="danger"
        onConfirm={handleDiscardConnectionGroup}
        onCancel={() => setShowDiscardConfirm(false)}
      />

      {currentConnectionGroup?.metadata?.id && (
        <EmitEventModal
          isOpen={showEmitEventModal}
          onClose={() => setShowEmitEventModal(false)}
          sourceId={currentConnectionGroup.metadata.id}
          sourceName={currentConnectionGroup.metadata.name || ''}
          sourceType="connection_group"
          lastEvent={currentConnectionGroup.events?.[0]}
        />
      )}
    </div>
  );
}
