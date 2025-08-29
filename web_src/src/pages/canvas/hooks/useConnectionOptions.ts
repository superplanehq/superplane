import { useMemo } from 'react';
import { useCanvasStore } from '../store/canvasStore';
import { SuperplaneConnectionType, SuperplaneConnection } from '@/api-client';

interface ConnectionOption {
  value: string;
  label: string;
  group: string;
  type: SuperplaneConnectionType;
}

export function useConnectionOptions(currentEntityId?: string, existingConnections?: SuperplaneConnection[]) {
  const { stages, eventSources, connectionGroups } = useCanvasStore();

  const getConnectionOptions = useMemo(() => {
    return (currentConnectionIndex?: number): ConnectionOption[] => {
      const options: ConnectionOption[] = [];
      const seenNames = new Set<string>();
      
      // Get names of existing connections (excluding the current one being edited)
      const existingConnectionNames = new Set(
        existingConnections
          ?.filter((_, index) => index !== currentConnectionIndex)
          ?.map(conn => conn.name)
          ?.filter(Boolean) || []
      );

      stages.forEach(stage => {
        if (stage.metadata?.name && 
            stage.metadata?.id !== currentEntityId && 
            !seenNames.has(stage.metadata.name) &&
            !existingConnectionNames.has(stage.metadata.name)) {
          seenNames.add(stage.metadata.name);
          options.push({
            value: stage.metadata.name,
            label: stage.metadata.name,
            group: 'Stages',
            type: 'TYPE_STAGE'
          });
        }
      });
      eventSources.forEach(eventSource => {
        if (eventSource.metadata?.name && 
            !seenNames.has(eventSource.metadata.name) &&
            !existingConnectionNames.has(eventSource.metadata.name)) {
          seenNames.add(eventSource.metadata.name);
          options.push({
            value: eventSource.metadata.name,
            label: eventSource.metadata.name,
            group: 'Event Sources',
            type: 'TYPE_EVENT_SOURCE'
          });
        }
      });
      connectionGroups.forEach(group => {
        if (group.metadata?.name && 
            group.metadata?.id !== currentEntityId && 
            !seenNames.has(group.metadata.name) &&
            !existingConnectionNames.has(group.metadata.name)) {
          seenNames.add(group.metadata.name);
          options.push({
            value: group.metadata.name,
            label: group.metadata.name,
            group: 'Connection Groups',
            type: 'TYPE_CONNECTION_GROUP'
          });
        }
      });
      return options;
    };
  }, [stages, eventSources, connectionGroups, currentEntityId, existingConnections]);

  return { getConnectionOptions };
}