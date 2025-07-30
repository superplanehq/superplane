import { useState } from 'react';
import { StageNodeType } from '@/canvas/types/flow';
import { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneValueDefinition, SuperplaneConnection, SuperplaneFilter, SuperplaneFilterOperator, SuperplaneFilterType, SuperplaneConnectionType, SuperplaneValueFrom } from '@/api-client/types.gen';
import { AccordionItem } from './AccordionItem';
import { Label } from './Label';
import { Field } from './Field';
import { Button } from '@/components/Button/button';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface EditModeContentProps {
  data: StageNodeType['data'];
  onSave: (editedData: { label: string; inputs: SuperplaneInputDefinition[]; outputs: SuperplaneOutputDefinition[] }) => void;
}

export function EditModeContent({ data, onSave }: EditModeContentProps) {
  const [openSections, setOpenSections] = useState<string[]>(['general']);
  const [inputs, setInputs] = useState<SuperplaneInputDefinition[]>(data.inputs || []);
  const [outputs, setOutputs] = useState<SuperplaneOutputDefinition[]>(data.outputs || []);
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [secrets, setSecrets] = useState<SuperplaneValueDefinition[]>([]);
  const [editingInputIndex, setEditingInputIndex] = useState<number | null>(null);
  const [editingOutputIndex, setEditingOutputIndex] = useState<number | null>(null);
  const [editingConnectionIndex, setEditingConnectionIndex] = useState<number | null>(null);
  const [editingSecretIndex, setEditingSecretIndex] = useState<number | null>(null);
  const [inputMappings, setInputMappings] = useState<Record<number, SuperplaneValueDefinition[]>>({});

  const handleAccordionToggle = (sectionId: string) => {
    setOpenSections(prev => {
      console.log(prev)
      return prev.includes(sectionId)
        ? prev.filter(id => id !== sectionId)
        : [...prev, sectionId]
    });
  };

  const addInput = () => {
    const newInput: SuperplaneInputDefinition = {
      name: '',
      description: ''
    };
    const newIndex = inputs.length;
    setInputs(prev => [...prev, newInput]);
    setEditingInputIndex(newIndex);
  };

  const updateInput = (index: number, field: keyof SuperplaneInputDefinition, value: string) => {
    setInputs(prev => prev.map((input, i) =>
      i === index ? { ...input, [field]: value } : input
    ));
  };

  const removeInput = (index: number) => {
    setInputs(prev => prev.filter((_, i) => i !== index));
    setEditingInputIndex(null);
  };

  const addOutput = () => {
    const newOutput: SuperplaneOutputDefinition = {
      name: '',
      description: '',
      required: false
    };
    const newIndex = outputs.length;
    setOutputs(prev => [...prev, newOutput]);
    setEditingOutputIndex(newIndex);
  };

  const updateOutput = (index: number, field: keyof SuperplaneOutputDefinition, value: string | boolean) => {
    setOutputs(prev => prev.map((output, i) =>
      i === index ? { ...output, [field]: value } : output
    ));
  };

  const removeOutput = (index: number) => {
    setOutputs(prev => prev.filter((_, i) => i !== index));
    setEditingOutputIndex(null);
  };

  const addMapping = (inputIndex: number) => {
    const newMapping: SuperplaneValueDefinition = {
      name: '',
      value: ''
    };
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: [...(prev[inputIndex] || []), newMapping]
    }));
  };

  const updateMapping = (inputIndex: number, mappingIndex: number, field: keyof SuperplaneValueDefinition, value: string) => {
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: prev[inputIndex]?.map((mapping, i) =>
        i === mappingIndex ? { ...mapping, [field]: value } : mapping
      ) || []
    }));
  };

  const removeMapping = (inputIndex: number, mappingIndex: number) => {
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: prev[inputIndex]?.filter((_, i) => i !== mappingIndex) || []
    }));
  };

  // Connection management
  const addConnection = () => {
    const newConnection: SuperplaneConnection = {
      name: '',
      type: 'TYPE_EVENT_SOURCE',
      filters: []
    };
    const newIndex = connections.length;
    setConnections(prev => [...prev, newConnection]);
    setEditingConnectionIndex(newIndex);
  };

  const updateConnection = (index: number, field: keyof SuperplaneConnection, value: SuperplaneConnectionType | SuperplaneFilterOperator | string) => {
    setConnections(prev => prev.map((conn, i) =>
      i === index ? { ...conn, [field]: value } : conn
    ));
  };

  const removeConnection = (index: number) => {
    setConnections(prev => prev.filter((_, i) => i !== index));
    setEditingConnectionIndex(null);
  };

  const addFilter = (connectionIndex: number) => {
    const newFilter: SuperplaneFilter = {
      type: 'FILTER_TYPE_DATA',
      data: { expression: '' }
    };

    setConnections(prev => prev.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: [...(conn.filters || []), newFilter]
      } : conn
    ));
  };

  const updateFilter = (connectionIndex: number, filterIndex: number, updates: Partial<SuperplaneFilter>) => {
    setConnections(prev => prev.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: conn.filters?.map((filter, j) =>
          j === filterIndex ? { ...filter, ...updates } : filter
        )
      } : conn
    ));
  };

  const removeFilter = (connectionIndex: number, filterIndex: number) => {
    setConnections(prev => prev.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: conn.filters?.filter((_, j) => j !== filterIndex)
      } : conn
    ));
  };

  const toggleFilterOperator = (connectionIndex: number) => {
    const current = connections[connectionIndex]?.filterOperator || 'FILTER_OPERATOR_AND';
    const newOperator: SuperplaneFilterOperator =
      current === 'FILTER_OPERATOR_AND' ? 'FILTER_OPERATOR_OR' : 'FILTER_OPERATOR_AND';

    updateConnection(connectionIndex, 'filterOperator', newOperator);
  };

  // Secret management
  const addSecret = () => {
    const newSecret: SuperplaneValueDefinition = {
      name: '',
      value: ''
    };
    const newIndex = secrets.length;
    setSecrets(prev => [...prev, newSecret]);
    setEditingSecretIndex(newIndex);
  };

  const updateSecret = (index: number, field: keyof SuperplaneValueDefinition, value: string | SuperplaneValueFrom) => {
    setSecrets(prev => prev.map((secret, i) =>
      i === index ? { ...secret, [field]: value } : secret
    ));
  };

  const updateSecretMode = (index: number, useValueFrom: boolean) => {
    setSecrets(prev => prev.map((secret, i) =>
      i === index ? {
        ...secret,
        value: useValueFrom ? undefined : '',
        valueFrom: useValueFrom ? { secret: { name: '', key: '' } } : undefined
      } : secret
    ));
  };

  const removeSecret = (index: number) => {
    setSecrets(prev => prev.filter((_, i) => i !== index));
    setEditingSecretIndex(null);
  };

  const handleSave = () => {
    onSave({
      label: data.label,
      inputs,
      outputs
    });
  };

  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      {/* Accordion Sections */}
      <div className="">

        {/* Connections Section */}
        <AccordionItem
          id="connections"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Connections</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {connections.length} connections
              </span>
            </div>
          }
          isOpen={openSections.includes('connections')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {connections.map((connection, index) => (
              <div key={index}>
                {editingConnectionIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Connection Type</Label>
                      <select
                        value={connection.type || 'TYPE_EVENT_SOURCE'}
                        onChange={(e) => updateConnection(index, 'type', e.target.value as SuperplaneConnectionType)}
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      >
                        <option value="TYPE_EVENT_SOURCE">Event Source</option>
                        <option value="TYPE_STAGE">Stage</option>
                        <option value="TYPE_CONNECTION_GROUP">Connection Group</option>
                      </select>
                    </Field>
                    <Field>
                      <Label>Connection Name</Label>
                      <input
                        type="text"
                        value={connection.name || ''}
                        onChange={(e) => updateConnection(index, 'name', e.target.value)}
                        placeholder="Connection name"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>

                    {/* Filters Section */}
                    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                      <div className="flex justify-between items-center mb-2">
                        <Label className="text-sm font-medium">Filters</Label>
                        <button
                          onClick={() => addFilter(index)}
                          className="text-blue-600 hover:text-blue-700 text-sm"
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
                                  const type = e.target.value as SuperplaneFilterType;
                                  const updates: Partial<SuperplaneFilter> = { type };
                                  if (type === 'FILTER_TYPE_DATA') {
                                    updates.data = { expression: filter.data?.expression || '' };
                                    updates.header = undefined;
                                  } else {
                                    updates.header = { expression: filter.header?.expression || '' };
                                    updates.data = undefined;
                                  }
                                  updateFilter(index, filterIndex, updates);
                                }}
                                className="px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
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
                                  updateFilter(index, filterIndex, updates);
                                }}
                                placeholder="Filter expression"
                                className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                              />
                              <button
                                onClick={() => removeFilter(index, filterIndex)}
                                className="text-red-600 hover:text-red-700"
                              >
                                <span className="material-symbols-outlined text-sm">delete</span>
                              </button>
                            </div>
                            {/* OR/AND toggle between filters */}
                            {filterIndex < (connection.filters?.length || 0) - 1 && (
                              <div className="flex justify-center py-1">
                                <button
                                  onClick={() => toggleFilterOperator(index)}
                                  className="px-3 py-1 text-xs bg-zinc-200 dark:bg-zinc-700 rounded-full hover:bg-zinc-300 dark:hover:bg-zinc-600"
                                >
                                  {connection.filterOperator === 'FILTER_OPERATOR_OR' ? 'OR' : 'AND'}
                                </button>
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => setEditingConnectionIndex(null)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingConnectionIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{connection.name || `Connection ${index + 1}`}</span>
                      {connection.type && (
                        <span className="text-xs bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-300 px-2 py-0.5 rounded">
                          {connection.type.replace('TYPE_', '').replace('_', ' ').toLowerCase()}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingConnectionIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeConnection(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <button
              onClick={addConnection}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Connection
            </button>
          </div>
        </AccordionItem>

        {/* Inputs Section */}
        <AccordionItem
          id="inputs"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Inputs</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {inputs.length} inputs
              </span>
            </div>
          }
          isOpen={openSections.includes('inputs')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {inputs.map((input, index) => (
              <div key={index}>
                {editingInputIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Name</Label>
                      <input
                        type="text"
                        value={input.name || ''}
                        onChange={(e) => updateInput(index, 'name', e.target.value)}
                        placeholder="Input name"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>
                    <Field>
                      <Label>Description</Label>
                      <textarea
                        value={input.description || ''}
                        onChange={(e) => updateInput(index, 'description', e.target.value)}
                        placeholder="Input description"
                        rows={2}
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>

                    {/* Mappings Section */}
                    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                      <div className="flex justify-between items-center mb-2">
                        <Label className="text-sm font-medium">Mappings</Label>
                        <button
                          onClick={() => addMapping(index)}
                          className="text-blue-600 hover:text-blue-700 text-sm"
                        >
                          + Add Mapping
                        </button>
                      </div>
                      <div className="space-y-2">
                        {(inputMappings[index] || []).map((mapping, mappingIndex) => (
                          <div key={mappingIndex} className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                            <select
                              value={mapping.name || ''}
                              onChange={(e) => updateMapping(index, mappingIndex, 'name', e.target.value)}
                              className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            >
                              <option value="">Select connection</option>
                              {data.connections.map((conn, connIndex) => (
                                <option key={connIndex} value={conn.name}>{conn.name}</option>
                              ))}
                            </select>
                            <select
                              value={mapping.value || ''}
                              onChange={(e) => updateMapping(index, mappingIndex, 'value', e.target.value)}
                              className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            >
                              <option value="">Select output value</option>
                              {data.outputs.map((output, outputIndex) => (
                                <option key={outputIndex} value={output.name}>{output.name}</option>
                              ))}
                            </select>
                            <button
                              onClick={() => removeMapping(index, mappingIndex)}
                              className="text-red-600 hover:text-red-700"
                            >
                              <span className="material-symbols-outlined text-sm">delete</span>
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => setEditingInputIndex(null)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingInputIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <span className="text-sm font-medium">{input.name || `Input ${index + 1}`}</span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingInputIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeInput(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <button
              onClick={addInput}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Input
            </button>
          </div>
        </AccordionItem>

        {/* Outputs Section */}
        <AccordionItem
          id="outputs"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Outputs</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {outputs.length} outputs
              </span>
            </div>
          }
          isOpen={openSections.includes('outputs')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {outputs.map((output, index) => (
              <div key={index}>
                {editingOutputIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Name</Label>
                      <input
                        type="text"
                        value={output.name || ''}
                        onChange={(e) => updateOutput(index, 'name', e.target.value)}
                        placeholder="Output name"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>
                    <Field>
                      <Label>Description</Label>
                      <textarea
                        value={output.description || ''}
                        onChange={(e) => updateOutput(index, 'description', e.target.value)}
                        placeholder="Output description"
                        rows={2}
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>
                    <Field>
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={`required-${index}`}
                          checked={output.required || false}
                          onChange={(e) => updateOutput(index, 'required', e.target.checked)}
                          className="w-4 h-4 text-blue-600 bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600 rounded focus:ring-blue-500"
                        />
                        <Label htmlFor={`required-${index}`}>Required</Label>
                      </div>
                    </Field>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => setEditingOutputIndex(null)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingOutputIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{output.name || `Output ${index + 1}`}</span>
                      {output.required && (
                        <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">Required</span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingOutputIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeOutput(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <button
              onClick={addOutput}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Output
            </button>
          </div>
        </AccordionItem>

        {/* Secrets Section */}
        <AccordionItem
          id="secrets"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Secrets Management</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {secrets.length} secrets
              </span>
            </div>
          }
          isOpen={openSections.includes('secrets')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {secrets.map((secret, index) => (
              <div key={index}>
                {editingSecretIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Secret Name</Label>
                      <input
                        type="text"
                        value={secret.name || ''}
                        onChange={(e) => updateSecret(index, 'name', e.target.value)}
                        placeholder="Secret name"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>

                    <Field>
                      <div className="flex items-center gap-4 mb-2">
                        <Label>Value Source</Label>
                        <div className="flex items-center gap-4">
                          <label className="flex items-center gap-2">
                            <input
                              type="radio"
                              name={`valueSource-${index}`}
                              checked={!secret.valueFrom}
                              onChange={() => updateSecretMode(index, false)}
                              className="w-4 h-4"
                            />
                            <span className="text-sm">Direct Value</span>
                          </label>
                          <label className="flex items-center gap-2">
                            <input
                              type="radio"
                              name={`valueSource-${index}`}
                              checked={!!secret.valueFrom}
                              onChange={() => updateSecretMode(index, true)}
                              className="w-4 h-4"
                            />
                            <span className="text-sm">From Secret</span>
                          </label>
                        </div>
                      </div>

                      {!secret.valueFrom ? (
                        <input
                          type="password"
                          value={secret.value || ''}
                          onChange={(e) => updateSecret(index, 'value', e.target.value)}
                          placeholder="Secret value"
                          className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      ) : (
                        <div className="space-y-2">
                          <input
                            type="text"
                            value={secret.valueFrom?.secret?.name || ''}
                            onChange={(e) => updateSecret(index, 'valueFrom', {
                              ...secret.valueFrom,
                              secret: { ...secret.valueFrom?.secret, name: e.target.value }
                            })}
                            placeholder="Secret name"
                            className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                          <input
                            type="text"
                            value={secret.valueFrom?.secret?.key || ''}
                            onChange={(e) => updateSecret(index, 'valueFrom', {
                              ...secret.valueFrom,
                              secret: { ...secret.valueFrom?.secret, key: e.target.value }
                            })}
                            placeholder="Secret key"
                            className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </div>
                      )}
                    </Field>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => setEditingSecretIndex(null)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingSecretIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <span className="text-sm font-medium">{secret.name || `Secret ${index + 1}`}</span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingSecretIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeSecret(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <button
              onClick={addSecret}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Secret
            </button>
          </div>
        </AccordionItem>
      </div>
    </div>
  );
}