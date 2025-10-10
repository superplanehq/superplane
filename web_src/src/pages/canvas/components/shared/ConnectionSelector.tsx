import { SuperplaneConnection, SuperplaneConnectionType, SuperplaneFilter } from '@/api-client/types.gen';
import { ValidationField } from '../../../../components/ValidationField';
import { useConnectionOptions } from '../../hooks/useConnectionOptions';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { StageFilterTooltip } from '@/components/Tooltip/StageFilterTooltip';
import { AutoCompleteSelect, type AutoCompleteOption } from '@/components/AutoCompleteSelect';
import { AutoCompleteInput } from '@/components/AutoCompleteInput/AutoCompleteInput';
import { validateBooleanExpression, FilterTypes } from '@/utils/exprValidator';
import { useState, useCallback, useImperativeHandle, forwardRef } from 'react';

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
  getEventTemplate?: (connectionName: string) => Record<string, unknown> | null;
}

interface FilterValidationState {
  isValid: boolean;
  isValidating: boolean;
  error?: string;
}

interface ConnectionSelectorRef {
  validateFilters: () => Promise<boolean>;
}

export const ConnectionSelector = forwardRef<ConnectionSelectorRef, ConnectionSelectorProps>(({
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
  existingConnections,
  getEventTemplate
}, ref) => {
  const { getConnectionOptions } = useConnectionOptions(currentEntityId, existingConnections);

  // State to track validation status for each filter
  const [filterValidation, setFilterValidation] = useState<Record<number, FilterValidationState>>({});

  // On-demand validation function for all filters
  const validateFilters = useCallback(async (): Promise<boolean> => {
    if (!connection.filters || connection.filters.length === 0) {
      return true;
    }

    let allValid = true;

    // Reset validation state and set all to validating
    const initialValidationState: Record<number, FilterValidationState> = {};
    connection.filters.forEach((_, index) => {
      initialValidationState[index] = { isValid: false, isValidating: true };
    });
    setFilterValidation(initialValidationState);

    // Validate each filter
    const validationPromises = connection.filters.map(async (filter, filterIndex) => {
      const expression = filter.type === 'FILTER_TYPE_HEADER'
        ? filter.header?.expression || ''
        : filter.data?.expression || '';

      // Empty expressions are considered valid
      if (!expression.trim()) {
        setFilterValidation(prev => ({
          ...prev,
          [filterIndex]: { isValid: true, isValidating: false }
        }));
        return true;
      }

      try {
        // Get event template for variables
        const eventTemplate = getEventTemplate ? getEventTemplate(connection.name || '') : {};
        const filterTypeMapping = filter.type === 'FILTER_TYPE_HEADER' ? FilterTypes.HEADER : FilterTypes.DATA;

        const result = await validateBooleanExpression(
          expression,
          eventTemplate || {},
          filterTypeMapping
        );

        const isValid = result.valid === true;

        setFilterValidation(prev => ({
          ...prev,
          [filterIndex]: {
            isValid,
            isValidating: false,
            error: result.error
          }
        }));

        return isValid;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Validation error';
        setFilterValidation(prev => ({
          ...prev,
          [filterIndex]: {
            isValid: false,
            isValidating: false,
            error: errorMessage
          }
        }));
        return false;
      }
    });

    const results = await Promise.all(validationPromises);
    allValid = results.every(result => result);

    return allValid;
  }, [connection.filters, connection.name, getEventTemplate]);

  // Expose validation function to parent component
  useImperativeHandle(ref, () => ({
    validateFilters
  }), [validateFilters]);

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
            Pro tip: Expressions are parsed using the <a className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300" href="https://expr-lang.org/docs/language-definition" target="_blank" rel="noopener noreferrer">Expr</a> language. You can type $ to select data from event payload.
          </div>
          <div className="space-y-2">
            {(connection.filters || []).map((filter, filterIndex) => {
              const hasError = filterErrors.includes(filterIndex + 1); // Convert to 1-based index for comparison
              const validation = filterValidation[filterIndex];
              const hasValidationError = validation && !validation.isValid && !validation.isValidating;
              const isValidating = validation && validation.isValidating;

              return (
                <div key={filterIndex}>
                  <div className={`flex gap-2 items-center p-2 rounded ${hasError || hasValidationError
                    ? 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700'
                    : validation && validation.isValid
                      ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700'
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
                      className={`px-2 py-1 border rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100 ${hasError || hasValidationError
                        ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600'
                        }`}
                    >
                      <option value="FILTER_TYPE_DATA">Data</option>
                      <option value="FILTER_TYPE_HEADER">Header</option>
                    </select>
                    <AutoCompleteInput
                      value={
                        filter.type === 'FILTER_TYPE_HEADER'
                          ? filter.header?.expression || ''
                          : filter.data?.expression || ''
                      }
                      onChange={(expression) => {
                        const updates: Partial<SuperplaneFilter> = {};
                        if (filter.type === 'FILTER_TYPE_HEADER') {
                          updates.header = { expression };
                        } else {
                          updates.data = { expression };
                        }
                        onFilterUpdate(index, filterIndex, updates);
                      }}
                      placeholder={filter.type === 'FILTER_TYPE_DATA' ? 'eg. $.execution.result=="passed"' : 'eg. headers["name"]=="value"'}
                      className={`flex-1 ${hasError || hasValidationError
                        ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                        : validation && validation.isValid
                          ? 'border-green-300 dark:border-green-600 focus:ring-green-500'
                          : 'border-zinc-300 dark:border-zinc-600'
                        }`}
                      inputSize="sm"
                      exampleObj={getEventTemplate ? getEventTemplate(connection.name || '') : {}}
                      startWord='$'
                      prefix='$.'
                      showValuePreview
                      noSuggestionsText="This component hasn't received any events from this connection. Send events to this connection to enable autocomplete suggestions."
                    />
                    {/* Validation status indicator */}
                    {isValidating && (
                      <div className="flex items-center text-blue-500" title="Validating expression...">
                        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-500"></div>
                      </div>
                    )}
                    {validation && validation.isValid && !isValidating && (
                      <div className="flex items-center text-green-500" title="Expression is valid">
                        <MaterialSymbol name="check_circle" size="sm" />
                      </div>
                    )}
                    {hasValidationError && (
                      <div className="flex items-center text-red-500" title={validation?.error || 'Invalid expression'}>
                        <MaterialSymbol name="error" size="sm" />
                      </div>
                    )}
                    <button
                      onClick={() => onFilterRemove(index, filterIndex)}
                      className="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
                    >
                      <span className="material-symbols-outlined text-sm!">delete</span>
                    </button>
                  </div>
                  {(hasError || hasValidationError) && (
                    <div className="mt-1 px-2 py-1 bg-red-100 dark:bg-red-800/30 border border-red-200 dark:border-red-700 rounded text-xs">
                      <div className="flex items-center gap-1 text-red-700 dark:text-red-300">
                        <MaterialSymbol name="error" size="sm" />
                        {hasError
                          ? `Filter ${filterIndex + 1} is incomplete - all filter fields must be filled`
                          : validation?.error || 'Expression validation failed'
                        }
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
});

ConnectionSelector.displayName = 'ConnectionSelector';