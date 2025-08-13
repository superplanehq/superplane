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

  const updateConnection = useCallback((index: number, type: SuperplaneConnectionType, name: string) => {
    setConnections(connections.map((conn, i) => {
      if (i === index) {
        const updatedConnection = { ...conn, type, name };
        return updatedConnection;
      }
      return conn;
    }));
  }, [connections, setConnections]);

  const updateFilterOperator = useCallback((index: number, operator: SuperplaneFilterOperator) => {
    setConnections(connections.map((conn, i) => {
      if (i === index) {
        const updatedConnection = { ...conn, filterOperator: operator };
        return updatedConnection;
      }
      return conn;
    }));
  }, [connections, setConnections]);

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

    updateFilterOperator(connectionIndex, newOperator);
  }, [connections, updateFilterOperator]);

  const validateConnection = useCallback((connection: SuperplaneConnection): string[] => {
    const errors: string[] = [];
    
    if (!connection.name || connection.name.trim() === '') {
      errors.push('Connection name is required');
    }
    
    if (!connection.type) {
      errors.push('Connection type is required');
    }
    
    if (connection.filters && connection.filters.length > 0) {
      const emptyFilters: number[] = [];
      
      connection.filters.forEach((filter, index) => {
        if (filter.type === 'FILTER_TYPE_DATA') {
          if (!filter.data?.expression || filter.data.expression.trim() === '') {
            emptyFilters.push(index + 1);
          }
        } else if (filter.type === 'FILTER_TYPE_HEADER') {
          if (!filter.header?.expression || filter.header.expression.trim() === '') {
            emptyFilters.push(index + 1);
          }
        }
      });
      
      if (emptyFilters.length > 0) {
        if (emptyFilters.length === 1) {
          errors.push(`Filter ${emptyFilters[0]} is incomplete - all filter fields must be filled`);
        } else {
          errors.push(`Filters ${emptyFilters.join(', ')} are incomplete - all filter fields must be filled`);
        }
      }
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