import { useCanvasStore } from '../store/canvasStore';

export const pollEventSourceUntilNoPending = async (canvasId: string, eventSourceId: string) => {
  const maxAttempts = 20;
  const pollInterval = 1000;
  let attempts = 0;

  const poll = async (): Promise<void> => {
    attempts++;
    await useCanvasStore.getState().syncEventSourceEvents(canvasId, eventSourceId);
    
    const currentEventSources = useCanvasStore.getState().eventSources;
    const eventSource = currentEventSources.find(es => es.metadata?.id === eventSourceId);
    
    if (!eventSource) {
      return;
    }

    const hasPendingEvents = eventSource.events?.some(event => event.state === 'STATE_PENDING') ?? false;
    
    if (hasPendingEvents && attempts < maxAttempts) {
      setTimeout(poll, pollInterval);
    }
  };

  await poll();
};