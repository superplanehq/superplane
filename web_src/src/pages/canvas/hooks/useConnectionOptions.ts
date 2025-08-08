import { useMemo } from 'react';
import { useCanvasStore } from '../store/canvasStore';

interface ConnectionOption {
  value: string;
  label: string;
  group: string;
}

export function useConnectionOptions(currentEntityId?: string) {
  const { stages, eventSources, connectionGroups } = useCanvasStore();

  const getConnectionOptions = useMemo(() => {
    return (): ConnectionOption[] => {
      const options: ConnectionOption[] = [];
      stages.forEach(stage => {
        if (stage.metadata?.name && stage.metadata?.id !== currentEntityId) {
          options.push({
            value: stage.metadata.name,
            label: stage.metadata.name,
            group: 'Stages'
          });
        }
      });
      eventSources.forEach(eventSource => {
        if (eventSource.metadata?.name) {
          options.push({
            value: eventSource.metadata.name,
            label: eventSource.metadata.name,
            group: 'Event Sources'
          });
        }
      });
      connectionGroups.forEach(group => {
        if (group.metadata?.name && group.metadata?.id !== currentEntityId) {
          options.push({
            value: group.metadata.name,
            label: group.metadata.name,
            group: 'Connection Groups'
          });
        }
      });
      return options;
    };
  }, [stages, eventSources, connectionGroups, currentEntityId]);

  return { getConnectionOptions };
}