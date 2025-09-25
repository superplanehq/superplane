import { SuperplaneConnection, SuperplaneConnectionType, SuperplaneFilter } from '@/api-client/types.gen';
import { ValidationField } from '../../../../components/ValidationField';
import { useConnectionOptions } from '../../hooks/useConnectionOptions';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { StageFilterTooltip } from '@/components/Tooltip/StageFilterTooltip';
import { AutoCompleteSelect, type AutoCompleteOption } from '@/components/AutoCompleteSelect';

interface ConnectionSelectorProps {
  connection: SuperplaneConnection;
  index: number;
  onConnectionUpdate: (index: number, type: SuperplaneConnectionType, name: string) => void;
  onFilterAdd: (connectionIndex: number) => void;
  onFilterUpdate: (connectionIndex: number, filterIndex: number, updates: Partial<SuperplaneFilter>) => void;
  onFilterRemove: (connectionIndex: number, filterIndex: number) => void;
  onFilterOperatorToggle: (connectionIndex: number) => void;
  currentEntityId?: string;
  validationError?: string;
  filterErrors?: number[]; // Array of filter indices that have validation errors
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
  filterErrors = [],
  showFilters = true,
  existingConnections
}: ConnectionSelectorProps) {
  const { getConnectionOptions } = useConnectionOptions(currentEntityId, existingConnections);

  const getAutoCompleteOptions = (): AutoCompleteOption[] => {
    const options = getConnectionOptions(index);

    return options.map(option => ({
      value: `${option.type}\u001F${option.value}`,
      label: option.label,
      group: option.group,
      type: option.type
    }));
  };

  return (
    <div className="space-y-3">
      <ValidationField
        label="Connection"
        error={validationError}
      >
        <AutoCompleteSelect
          options={getAutoCompleteOptions()}
          value={`${connection.type}\u001F${connection.name}`}
          onChange={(value) => {
            const [type, name] = value.split('\u001F');
            onConnectionUpdate(index, type as SuperplaneConnectionType, name)
          }}
          placeholder={connection.type ? 'Select a connection...' : 'Select connection type first'}
          error={!!validationError}
        />
      </ValidationField>

      {/* Filters Section */}
      {showFilters && (
        <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
          <div className="flex justify-between items-center mb-2">
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-900 dark:text-zinc-100">Filters</label>
              <StageFilterTooltip />
            </div>
          </div>
          <div className="mb-3 text-xs text-zinc-600 dark:text-zinc-400">
            Pro tip: Expressions are parsed using the <a className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300" href="https://expr-lang.org/docs/language-definition" target="_blank" rel="noopener noreferrer">Expr</a> language.
          </div>
          <div className="space-y-2">
            {(connection.filters || []).map((filter, filterIndex) => {
              const hasError = filterErrors.includes(filterIndex + 1); // Convert to 1-based index for comparison

              return (
                <div key={filterIndex}>
                  <div className={`flex gap-2 items-center p-2 rounded ${hasError
                    ? 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700'
                    : 'bg-zinc-50 dark:bg-zinc-800'
                    }`}>
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
                      className={`px-2 py-1 border rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100 ${hasError
                        ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600'
                        }`}
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
                      placeholder={filter.type === 'FILTER_TYPE_DATA' ? 'eg. $.execution.result=="passed"' : 'eg. headers["name"]=="value"'}
                      className={`flex-1 px-2 py-1 border rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100 ${hasError
                        ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600'
                        }`}
                    />
                    <button
                      onClick={() => onFilterRemove(index, filterIndex)}
                      className="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
                    >
                      <span className="material-symbols-outlined text-sm!">delete</span>
                    </button>
                  </div>
                  {hasError && (
                    <div className="mt-1 px-2 py-1 bg-red-100 dark:bg-red-800/30 border border-red-200 dark:border-red-700 rounded text-xs">
                      <div className="flex items-center gap-1 text-red-700 dark:text-red-300">
                        <MaterialSymbol name="error" size="sm" />
                        Filter {filterIndex + 1} is incomplete - all filter fields must be filled
                      </div>
                    </div>
                  )}
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
              );
            })}
          </div>
          <button
            onClick={() => onFilterAdd(index)}
            className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <MaterialSymbol name="add" size="sm" />
            Add Filter
          </button>
        </div>
      )}
    </div>
  );
}