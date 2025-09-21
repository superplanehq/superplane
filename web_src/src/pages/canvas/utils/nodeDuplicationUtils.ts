import { ConnectionGroupWithEvents, EventSourceWithEvents, Stage } from '../store/types';

/**
 * Generates a unique name for a duplicated entity by adding "-copy" suffix
 * and a number if needed to avoid conflicts
 */
export const generateUniqueNameCopy = (baseName: string, existingNames: string[]): string => {
  const normalizedExistingNames = existingNames.map(name => name.toLowerCase());
  let copyName = `${baseName}-copy`;
  let copyNumber = 1;

  if (normalizedExistingNames.includes(copyName.toLowerCase())) {
    while (normalizedExistingNames.includes(`${copyName}-${copyNumber}`.toLowerCase())) {
      copyNumber++;
    }
    copyName = `${copyName}-${copyNumber}`;
  }

  return copyName;
};

/**
 * Creates a duplicate of a stage with a new temporary ID and unique name
 */
export const createStageDuplicate = (originalStage: Stage, allStages: Stage[]): Stage => {
  const allStageNames = allStages.map(stage => stage.metadata?.name || '');
  const duplicateName = generateUniqueNameCopy(originalStage.metadata?.name || 'Stage', allStageNames);

  return {
    ...originalStage,
    metadata: {
      ...originalStage.metadata,
      id: Date.now().toString(), // Temporary ID
      name: duplicateName,
    },
    queue: [],
    executions: [],
    isDraft: true
  };
};

/**
 * Creates a duplicate of an event source with a new temporary ID and unique name
 */
export const createEventSourceDuplicate = (
  originalEventSource: EventSourceWithEvents,
  allEventSources: EventSourceWithEvents[]
): EventSourceWithEvents => {
  const allEventSourceNames = allEventSources.map(es => es.metadata?.name || '');
  const duplicateName = generateUniqueNameCopy(originalEventSource.metadata?.name || 'Event Source', allEventSourceNames);

  return {
    ...originalEventSource,
    metadata: {
      ...originalEventSource.metadata,
      id: Date.now().toString(), // Temporary ID
      name: duplicateName,
    },
    events: []
  };
};

/**
 * Creates a duplicate of a connection group with a new temporary ID and unique name
 */
export const createConnectionGroupDuplicate = (
  originalConnectionGroup: ConnectionGroupWithEvents,
  allConnectionGroups: ConnectionGroupWithEvents[]
): ConnectionGroupWithEvents => {
  const allConnectionGroupNames = allConnectionGroups.map(cg => cg.metadata?.name || '');
  const duplicateName = generateUniqueNameCopy(originalConnectionGroup.metadata?.name || 'Connection Group', allConnectionGroupNames);

  return {
    ...originalConnectionGroup,
    metadata: {
      ...originalConnectionGroup.metadata,
      id: Date.now().toString(), // Temporary ID
      name: duplicateName,
    },
    events: []
  };
};

/**
 * Generic function to focus and edit a newly created duplicate node
 */
export const focusAndEditNode = (
  nodeId: string,
  setFocusedNodeId: (id: string) => void,
  setEditingFunction: (id: string) => void
): void => {
  setTimeout(() => {
    setFocusedNodeId(nodeId);
    setEditingFunction(nodeId);
  }, 100);
};