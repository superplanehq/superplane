import { useCallback } from 'react';
import { SuperplaneConnection, SuperplaneConnectionType, SuperplaneFilter, SuperplaneFilterOperator } from '@/api-client/types.gen';
import { useConnectionOptions } from './useConnectionOptions';

interface UseConnectionManagerProps {
  connections: SuperplaneConnection[];
  setConnections: (connections: SuperplaneConnection[]) => void;
  currentEntityId?: string;
}

export function useConnectionManager({ connections, setConnections, currentEntityId }: UseConnectionManagerProps) {
  const { getConnectionOptions } = useConnectionOptions(currentEntityId);

  const updateConnection = useCallback((index: number, field: keyof SuperplaneConnection, value: SuperplaneConnectionType | SuperplaneFilterOperator | string) => {
    setConnections(connections.map((conn, i) => {
      if (i === index) {
        const updatedConnection = { ...conn, [field]: value };

        // If connection type changed, clear the connection name since available options will be different
        if (field === 'type' && updatedConnection.name) {
          const newOptions = getConnectionOptions(value as SuperplaneConnectionType);
          const isCurrentNameValid = newOptions.some(option => option.value === updatedConnection.name);
          if (!isCurrentNameValid) {
            updatedConnection.name = '';
          }
        }

        return updatedConnection;
      }
      return conn;
    }));
  }, [connections, setConnections, getConnectionOptions]);

  const addFilter = useCallback((connectionIndex: number) => {
    const newFilter: SuperplaneFilter = {
      type: 'FILTER_TYPE_DATA',
      data: { expression: '' }
    };

    setConnections(connections.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: [...(conn.filters || []), newFilter]
      } : conn
    ));
  }, [connections, setConnections]);

  const updateFilter = useCallback((connectionIndex: number, filterIndex: number, updates: Partial<SuperplaneFilter>) => {
    setConnections(connections.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: conn.filters?.map((filter, j) =>
          j === filterIndex ? { ...filter, ...updates } : filter
        )
      } : conn
    ));
  }, [connections, setConnections]);

  const removeFilter = useCallback((connectionIndex: number, filterIndex: number) => {
    setConnections(connections.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: conn.filters?.filter((_, j) => j !== filterIndex)
      } : conn
    ));
  }, [connections, setConnections]);

  const toggleFilterOperator = useCallback((connectionIndex: number) => {
    const current = connections[connectionIndex]?.filterOperator || 'FILTER_OPERATOR_AND';
    const newOperator: SuperplaneFilterOperator =
      current === 'FILTER_OPERATOR_AND' ? 'FILTER_OPERATOR_OR' : 'FILTER_OPERATOR_AND';

    updateConnection(connectionIndex, 'filterOperator', newOperator);
  }, [connections, updateConnection]);

  const validateConnection = useCallback((connection: SuperplaneConnection): string[] => {
    const errors: string[] = [];
    if (!connection.name || connection.name.trim() === '') {
      errors.push('Connection name is required');
    }
    if (!connection.type) {
      errors.push('Connection type is required');
    }
    return errors;
  }, []);

  return {
    updateConnection,
    addFilter,
    updateFilter,
    removeFilter,
    toggleFilterOperator,
    validateConnection,
    getConnectionOptions
  };
}