import { useState } from 'react';
import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { ConnectionGroupNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { useCreateConnectionGroup, useUpdateConnectionGroup } from '@/hooks/useCanvasData';
import { SuperplaneConnection, GroupByField, SpecTimeoutBehavior, SuperplaneConnectionGroup } from '@/api-client';
import { ConnectionGroupEditModeContent } from '../ConnectionGroupEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { EditModeActionButtons } from '../EditModeActionButtons';

export default function ConnectionGroupNode(props: NodeProps<ConnectionGroupNodeType>) {

  const isNewNode = props.id && /^\d+$/.test(props.id);
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ name: string; description?: string; connections: SuperplaneConnection[]; groupByFields: GroupByField[]; timeout?: number; timeoutBehavior?: SpecTimeoutBehavior; isValid: boolean } | null>(null);
  const [connectionGroupName, setConnectionGroupName] = useState(props.data.name);
  const [connectionGroupDescription, setConnectionGroupDescription] = useState(props.data.description || '');
  const { updateConnectionGroup, setEditingConnectionGroup, removeConnectionGroup } = useCanvasStore();

  const currentConnectionGroup = useCanvasStore(state =>
    state.connectionGroups.find(cg => cg.metadata?.id === props.id)
  );


  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const createConnectionGroupMutation = useCreateConnectionGroup(canvasId);
  const updateConnectionGroupMutation = useUpdateConnectionGroup(canvasId);
  const focusedNodeId = useCanvasStore(state => state.focusedNodeId);


  const groupByFields = props.data.groupBy?.fields || [];


  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingConnectionGroup(props.id);

    setConnectionGroupName(props.data.name);
    setConnectionGroupDescription(props.data.description || '');
  };

  const handleSaveConnectionGroup = async (saveAsDraft = false) => {
    if (!currentFormData || !currentConnectionGroup) {
      return;
    }

    const isTemporaryId = currentConnectionGroup.metadata?.id && /^\d+$/.test(currentConnectionGroup.metadata.id);
    const isNewConnectionGroup = !currentConnectionGroup.metadata?.id || isTemporaryId;

    try {
      if (isNewConnectionGroup && !saveAsDraft) {

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
      } else if (!isNewConnectionGroup && !saveAsDraft) {

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
      } else if (saveAsDraft) {

        const draftConnectionGroup = {
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
        };
        updateConnectionGroup(draftConnectionGroup);

        props.data.name = connectionGroupName;
        props.data.description = connectionGroupDescription;
      }
      setIsEditMode(false);
      setEditingConnectionGroup(null);
      setCurrentFormData(null);
    } catch (error) {
      console.error(`Failed to ${isNewConnectionGroup ? 'create' : 'update'} connection group:`, error);
    }
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingConnectionGroup(null);
    setCurrentFormData(null);

    setConnectionGroupName(props.data.name);
    setConnectionGroupDescription(props.data.description || '');
  };

  const handleDiscardConnectionGroup = () => {
    if (currentConnectionGroup?.metadata?.id) {
      removeConnectionGroup(currentConnectionGroup.metadata.id);
    }
    setShowDiscardConfirm(false);
  };

  const handleConnectionGroupNameChange = (newName: string) => {
    setConnectionGroupName(newName);
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

  const handleYamlApply = (updatedData: unknown) => {
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
  };

  return (
    <div
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative`}
      style={{ width: '390px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {focusedNodeId === props.id && (
        <EditModeActionButtons
          onSave={handleSaveConnectionGroup}
          onCancel={handleCancelEdit}
          onDiscard={() => setShowDiscardConfirm(true)}
          onEdit={handleEditClick}
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
      <div className="px-4 py-4 flex justify-between items-start">
        <div className="flex items-start flex-1 min-w-0">
          <span className="material-symbols-outlined mr-2 text-gray-700 mt-1">account_tree</span>
          <div className="flex-1 min-w-0">
            <div className="mb-1">
              <InlineEditable
                value={connectionGroupName}
                onSave={handleConnectionGroupNameChange}
                placeholder="Connection group name"
                className="font-bold text-gray-900 text-base text-left px-2 py-1"
                isEditMode={isEditMode}
              />
            </div>
            <div>
              <InlineEditable
                value={connectionGroupDescription}
                onSave={handleConnectionGroupDescriptionChange}
                placeholder={isEditMode ? "Add description..." : "No description available"}
                className="text-gray-600 text-sm text-left px-2 py-1"
                isEditMode={isEditMode}
              />
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2 ml-2">
          {!isEditMode && (
            <button
              onClick={handleEditClick}
              className="p-1 text-gray-500 hover:text-gray-700 transition-colors"
              title="Edit connection group"
            >
              <MaterialSymbol name="edit" size="md" />
            </button>
          )}
        </div>
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
          onDataChange={setCurrentFormData}
        />
      ) : (
        <>
          {/* Group By Section */}
          <div className="px-3 py-3 border-t w-full">
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-xs font-medium text-gray-700 uppercase tracking-wide">Group By Fields</div>
              <div className="text-xs text-gray-600">
                {groupByFields.length} fields
              </div>
            </div>

            <div className="space-y-1">
              {groupByFields.length > 0 ? (
                groupByFields.map((field, index) => (
                  <div key={index} className="bg-gray-100 rounded p-2">
                    <div className="flex justify-start items-center gap-3 overflow-hidden">
                      <span className="text-sm text-gray-600">
                        <i className="material-icons f3 fill-black rounded-full bg-[var(--washed-blue)] black-60 p-1">label</i>
                      </span>
                      <span className="truncate font-medium">{field.name}</span>
                      <span className="text-xs text-gray-500">→</span>
                      <span className="truncate text-sm">{field.expression}</span>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-500 italic py-2">No group by fields configured</div>
              )}
            </div>
          </div>

          {/* Connections Section */}
          <div className="px-3 py-3 border-t w-full">
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-xs font-medium text-gray-700 uppercase tracking-wide">Connections</div>
              <div className="text-xs text-gray-600">
                {props.data.connections?.length || 0} connections
              </div>
            </div>

            <div className="space-y-1">
              {props.data.connections?.length ? (
                props.data.connections.map((connection, index) => (
                  <div key={index} className="bg-gray-100 rounded p-2">
                    <div className="flex justify-start items-center gap-3 overflow-hidden">
                      <span className="text-sm text-gray-600">
                        <i className="material-icons f3 fill-black rounded-full bg-[var(--washed-green)] black-60 p-1">link</i>
                      </span>
                      <span className="truncate font-medium">{connection.name}</span>
                      <span className="text-xs bg-zinc-100 text-zinc-600 px-2 py-0.5 rounded">
                        {connection.type?.replace('TYPE_', '').replace('_', ' ').toLowerCase()}
                      </span>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-500 italic py-2">No connections configured</div>
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
    </div>
  );
}
