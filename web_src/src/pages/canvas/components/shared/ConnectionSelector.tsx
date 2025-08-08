import { SuperplaneConnection, SuperplaneConnectionType, SuperplaneFilter } from '@/api-client/types.gen';
import { ValidationField } from './ValidationField';
import { useConnectionOptions } from '../../hooks/useConnectionOptions';

interface ConnectionSelectorProps {
  connection: SuperplaneConnection;
  index: number;
  onConnectionUpdate: (index: number, field: keyof SuperplaneConnection, value: string | SuperplaneConnectionType) => void;
  onFilterAdd: (connectionIndex: number) => void;
  onFilterUpdate: (connectionIndex: number, filterIndex: number, updates: Partial<SuperplaneFilter>) => void;
  onFilterRemove: (connectionIndex: number, filterIndex: number) => void;
  onFilterOperatorToggle: (connectionIndex: number) => void;
  currentEntityId?: string;
  validationError?: string;
  showFilters?: boolean;
  existingConnections?: SuperplaneConnection[];
}

export function ConnectionSelector({
  connection,
  index,
  onConnectionUpdate,
  onFilterAdd,
  onFilterUpdate,
  onFilterRemove,
  onFilterOperatorToggle,
  currentEntityId,
  validationError,
  showFilters = true,
  existingConnections = []
}: ConnectionSelectorProps) {
  const { getConnectionOptions } = useConnectionOptions(currentEntityId);

  const renderBreadcrumbConnectionOptions = () => {
    // Get all connection options (not filtered by type)
    const allOptions = getConnectionOptions(undefined);

    // Filter out already added connections, but keep the current connection if it exists
    const filteredOptions = allOptions.filter(option => {
      // Check if this connection is already in the existingConnections array
      const isAlreadyAdded = existingConnections.some((existingConn, existingIndex) => {
        // Skip the current connection being edited
        if (existingIndex === index) {
          return false;
        }
        return existingConn.name === option.value;
      });

      return !isAlreadyAdded;
    });

    if (filteredOptions.length === 0) {
      return (
        <option value="" disabled>
          No available connections
        </option>
      );
    }

    const groupedOptions: Record<string, typeof filteredOptions> = {};

    // Group options by their group property
    filteredOptions.forEach(option => {
      if (!groupedOptions[option.group]) {
        groupedOptions[option.group] = [];
      }
      groupedOptions[option.group].push(option);
    });

    return Object.entries(groupedOptions).map(([groupName, groupOptions]) => (
      <optgroup key={groupName} label={groupName}>
        {groupOptions.map(option => (
          <option key={`${groupName}-${option.value}`} value={`${groupName}-${option.value}`}>
            {groupName.replace('s', '')} → {option.label}
          </option>
        ))}
      </optgroup>
    ));
  };

  const handleBreadcrumbConnectionChange = (value: string) => {
    if (!value) {
      onConnectionUpdate(index, 'type', 'TYPE_EVENT_SOURCE');
      onConnectionUpdate(index, 'name', '');
      return;
    }

    const [groupName, connectionName] = value.split('-');
    let connectionType: SuperplaneConnectionType;

    switch (groupName) {
      case 'Event Sources':
        connectionType = 'TYPE_EVENT_SOURCE';
        break;
      case 'Stages':
        connectionType = 'TYPE_STAGE';
        break;
      case 'Connection Groups':
        connectionType = 'TYPE_CONNECTION_GROUP';
        break;
      default:
        connectionType = 'TYPE_EVENT_SOURCE';
    }

    onConnectionUpdate(index, 'type', connectionType);
    onConnectionUpdate(index, 'name', connectionName);
  };

  const getCurrentBreadcrumbValue = () => {
    if (!connection.type || !connection.name) {
      return '';
    }

    let groupName = '';
    switch (connection.type) {
      case 'TYPE_EVENT_SOURCE':
        groupName = 'Event Sources';
        break;
      case 'TYPE_STAGE':
        groupName = 'Stages';
        break;
      case 'TYPE_CONNECTION_GROUP':
        groupName = 'Connection Groups';
        break;
      default:
        groupName = 'Event Sources';
    }

    return `${groupName}-${connection.name}`;
  };

  return (
    <div className="space-y-3">
      <ValidationField
        label="Connection"
        error={validationError}
      >
        <select
          value={getCurrentBreadcrumbValue()}
          onChange={(e) => handleBreadcrumbConnectionChange(e.target.value)}
          className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationError
            ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
            : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
            }`}
        >
          <option value="">Select a connection...</option>
          {renderBreadcrumbConnectionOptions()}
        </select>
      </ValidationField>

      {/* Filters Section */}
      {showFilters && (
        <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
          <div className="flex justify-between items-center mb-2">
            <label className="text-sm font-medium text-gray-900 dark:text-zinc-100">Filters</label>
            <button
              onClick={() => onFilterAdd(index)}
              className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 text-sm"
            >
              + Add Filter
            </button>
          </div>
          <div className="space-y-2">
            {(connection.filters || []).map((filter, filterIndex) => (
              <div key={filterIndex}>
                <div className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                  <select
                    value={filter.type || 'FILTER_TYPE_DATA'}
                    onChange={(e) => {
                      const type = e.target.value as SuperplaneFilter['type'];
                      const updates: Partial<SuperplaneFilter> = { type };
                      if (type === 'FILTER_TYPE_DATA') {
                        updates.data = { expression: filter.data?.expression || '' };
                        updates.header = undefined;
                      } else {
                        updates.header = { expression: filter.header?.expression || '' };
                        updates.data = undefined;
                      }
                      onFilterUpdate(index, filterIndex, updates);
                    }}
                    className="px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100"
                  >
                    <option value="FILTER_TYPE_DATA">Data</option>
                    <option value="FILTER_TYPE_HEADER">Header</option>
                  </select>
                  <input
                    type="text"
                    value={
                      filter.type === 'FILTER_TYPE_HEADER'
                        ? filter.header?.expression || ''
                        : filter.data?.expression || ''
                    }
                    onChange={(e) => {
                      const expression = e.target.value;
                      const updates: Partial<SuperplaneFilter> = {};
                      if (filter.type === 'FILTER_TYPE_HEADER') {
                        updates.header = { expression };
                      } else {
                        updates.data = { expression };
                      }
                      onFilterUpdate(index, filterIndex, updates);
                    }}
                    placeholder="Filter expression"
                    className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100"
                  />
                  <button
                    onClick={() => onFilterRemove(index, filterIndex)}
                    className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                  >
                    <span className="material-symbols-outlined text-sm">delete</span>
                  </button>
                </div>
                {/* OR/AND toggle between filters */}
                {filterIndex < (connection.filters?.length || 0) - 1 && (
                  <div className="flex justify-center py-1">
                    <button
                      onClick={() => onFilterOperatorToggle(index)}
                      className="px-3 py-1 text-xs bg-zinc-200 dark:bg-zinc-700 text-gray-900 dark:text-zinc-100 rounded-full hover:bg-zinc-300 dark:hover:bg-zinc-600"
                    >
                      {connection.filterOperator === 'FILTER_OPERATOR_OR' ? 'OR' : 'AND'}
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}