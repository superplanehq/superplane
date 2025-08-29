import { useCanvasStore } from '../store/canvasStore';
import { ConnectionGroupWithEvents, EventSourceWithEvents, StageWithEventQueue } from '../store/types';

const buildPollFunction = (sourceType: 'event-source' | 'connection-group' | 'stage', canvasId: string, sourceId: string, syncFunction: (canvasId: string, sourceId: string) => Promise<void>) => {
  const maxAttempts = 20;
  const pollInterval = 1000;
  let attempts = 0;

  const poll = async (): Promise<void> => {
    attempts++;
    await syncFunction(canvasId, sourceId);
    
    let currentSources: EventSourceWithEvents[] | ConnectionGroupWithEvents[] | StageWithEventQueue[] = []
    
    if (sourceType === 'event-source') {
      currentSources = useCanvasStore.getState().eventSources;
    } else if (sourceType === 'connection-group') {
      currentSources = useCanvasStore.getState().connectionGroups;
    } else if (sourceType === 'stage') {
      currentSources = useCanvasStore.getState().stages;
    }
    
    const source = currentSources.find(s => s.metadata?.id === sourceId);
    
    if (!source) {
      return;
    }

    const hasPendingEvents = source.events?.some(event => event.state === 'STATE_PENDING') ?? false;
    
    if (hasPendingEvents && attempts < maxAttempts) {
      setTimeout(poll, pollInterval);
    }
  };

  return poll;
};

export const pollEventSourceUntilNoPending = async (canvasId: string, eventSourceId: string) => {
  const poll = buildPollFunction('event-source', canvasId, eventSourceId, useCanvasStore.getState().syncEventSourceEvents);
  await poll();
};

export const pollConnectionGroupUntilNoPending = async (canvasId: string, connectionGroupId: string) => {
  const poll = buildPollFunction('connection-group', canvasId, connectionGroupId, useCanvasStore.getState().syncConnectionGroupPlainEvents);
  await poll();
};

export const pollStageUntilNoPending = async (canvasId: string, stageId: string) => {
  const poll = buildPollFunction('stage', canvasId, stageId, useCanvasStore.getState().syncStagePlainEvents);
  await poll();
};
    