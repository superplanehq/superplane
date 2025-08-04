import { useMemo } from 'react';
import { SuperplaneConnectionType } from '@/api-client/types.gen';
import { useCanvasStore } from '../store/canvasStore';

interface ConnectionOption {
  value: string;
  label: string;
  group: string;
}

export function useConnectionOptions(currentEntityId?: string) {
  const { stages, eventSources, connectionGroups } = useCanvasStore();

  const getConnectionOptions = useMemo(() => {
    return (connectionType: SuperplaneConnectionType | undefined): ConnectionOption[] => {
      const options: ConnectionOption[] = [];

      switch (connectionType) {
        case 'TYPE_STAGE':
          stages.forEach(stage => {
            if (stage.metadata?.name && stage.metadata?.id !== currentEntityId) {
              options.push({
                value: stage.metadata.name,
                label: stage.metadata.name,
                group: 'Stages'
              });
            }
          });
          break;

        case 'TYPE_EVENT_SOURCE':
          eventSources.forEach(eventSource => {
            if (eventSource.metadata?.name) {
              options.push({
                value: eventSource.metadata.name,
                label: eventSource.metadata.name,
                group: 'Event Sources'
              });
            }
          });
          break;

        case 'TYPE_CONNECTION_GROUP':
          connectionGroups.forEach(group => {
            if (group.metadata?.name && group.metadata?.id !== currentEntityId) {
              options.push({
                value: group.metadata.name,
                label: group.metadata.name,
                group: 'Connection Groups'
              });
            }
          });
          break;

        default:
          // If no type selected, show all available connections
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
      }

      return options;
    };
  }, [stages, eventSources, connectionGroups, currentEntityId]);

  return { getConnectionOptions };
}