import { useState } from 'react';
import { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { ConnectionGroupNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { useCreateConnectionGroup, useUpdateConnectionGroup } from '@/hooks/useCanvasData';
import { SuperplaneConnection, GroupByField, SpecTimeoutBehavior } from '@/api-client';
import { ConnectionGroupEditModeContent } from '../ConnectionGroupEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Dropdown, DropdownButton, DropdownItem, DropdownLabel, DropdownMenu } from '@/components/Dropdown/dropdown';

export default function ConnectionGroupNode(props: NodeProps<ConnectionGroupNodeType>) {
  // Check if this is a newly added node (has temporary ID)
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

  // Get canvasId from the store or current context
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const createConnectionGroupMutation = useCreateConnectionGroup(canvasId);
  const updateConnectionGroupMutation = useUpdateConnectionGroup(canvasId);

  // Extract group by fields for display
  const groupByFields = props.data.groupBy?.fields || [];

  // Edit mode handlers
  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingConnectionGroup(props.id);
    // Initialize the editable values from current data
    setConnectionGroupName(props.data.name);
    setConnectionGroupDescription(props.data.description || '');
  };

  const handleSaveConnectionGroup = async (saveAsDraft = false) => {
    if (!currentFormData || !currentConnectionGroup) {
      return;
    }

    // Check if this is a new/draft connection group
    const isTemporaryId = currentConnectionGroup.metadata?.id && /^\d+$/.test(currentConnectionGroup.metadata.id);
    const isNewConnectionGroup = !currentConnectionGroup.metadata?.id || isTemporaryId;

    try {
      if (isNewConnectionGroup && !saveAsDraft) {
        // Create new connection group (commit to backend)
        const result = await createConnectionGroupMutation.mutateAsync({
          name: connectionGroupName,
          description: connectionGroupDescription,
          connections: currentFormData.connections,
          groupByFields: currentFormData.groupByFields,
          timeout: currentFormData.timeout,
          timeoutBehavior: currentFormData.timeoutBehavior
        });

        // Update local store with the new connection group data from API response
        const newConnectionGroup = result.data?.connectionGroup;
        if (newConnectionGroup) {
          removeConnectionGroup(props.id);
        }
      } else if (!isNewConnectionGroup && !saveAsDraft) {
        // Update existing connection group (commit to backend)
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

        // Update local store as well
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
        // Update props.data to reflect the changes
        props.data.name = connectionGroupName;
        props.data.description = connectionGroupDescription;
      } else if (saveAsDraft) {
        // Save as draft (only update local store, don't commit to backend)
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
        // Update props.data to reflect the changes
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
    // Reset to original values
    setConnectionGroupName(props.data.name);
    setConnectionGroupDescription(props.data.description || '');
  };

  // Check if this is a draft/new connection group that can be discarded
  const isDraftConnectionGroup = () => {
    if (!currentConnectionGroup) return false;

    // Check if it has a temporary ID (timestamp strings)
    const isTemporaryId = currentConnectionGroup.metadata?.id && /^\d+$/.test(currentConnectionGroup.metadata.id);

    // Check if it doesn't have an ID yet
    const hasNoId = !currentConnectionGroup.metadata?.id;

    return isTemporaryId || hasNoId;
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

  return (
    <div
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative`}
      style={{ width: '390px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {isEditMode && (
        <div
          className="action-buttons absolute z-50 -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white shadow-lg rounded-lg px-2 py-1 border border-gray-200 z-50"
          onClick={(e) => e.stopPropagation()}
        >
          <Dropdown>
            <DropdownButton plain className='flex items-center gap-2'>
              <MaterialSymbol name="save" size="md" />
              Save
              <MaterialSymbol name="expand_more" size="md" />
            </DropdownButton>
            <DropdownMenu anchor="bottom start">
              <DropdownItem className='flex items-center gap-2' onClick={() => handleSaveConnectionGroup(false)}>
                <DropdownLabel>Save & Commit</DropdownLabel>
              </DropdownItem>
              <DropdownItem className='flex items-center gap-2' onClick={() => handleSaveConnectionGroup(true)}>
                <DropdownLabel>Save as Draft</DropdownLabel>
              </DropdownItem>
            </DropdownMenu>
          </Dropdown>

          <button
            onClick={handleCancelEdit}
            className="flex items-center gap-2 px-3 py-2 text-gray-600 hover:text-gray-800 hover:bg-gray-50 rounded-md transition-colors"
            title="Cancel changes"
          >
            <MaterialSymbol name="close" size="md" />
            Cancel
          </button>

          {isDraftConnectionGroup() && (
            <button
              onClick={() => setShowDiscardConfirm(true)}
              className="flex items-center gap-2 px-3 py-2 text-red-600 hover:text-red-800 hover:bg-red-50 rounded-md transition-colors"
              title="Discard this connection group"
            >
              <MaterialSymbol name="delete" size="md" />
              Discard
            </button>
          )}
        </div>
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
          data={props.data}
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
