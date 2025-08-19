import { SuperplaneInputMapping, SuperplaneInputDefinition } from '@/api-client';

export interface InputMappingHandlersParams {
  inputMappings: SuperplaneInputMapping[];
  setInputMappings: (mappings: SuperplaneInputMapping[]) => void;
  inputs: SuperplaneInputDefinition[];
}

export const createInputMappingHandlers = ({
  inputMappings,
  setInputMappings,
  inputs
}: InputMappingHandlersParams) => {
  
  const handleAddMapping = (input: SuperplaneInputDefinition) => {
    const existingEmptyMappingIndex = inputMappings.findIndex(mapping =>
      !mapping.when?.triggeredBy?.connection || mapping.when.triggeredBy.connection === ''
    );

    if (existingEmptyMappingIndex !== -1) {
      const newMappings = [...inputMappings];
      const existingValues = newMappings[existingEmptyMappingIndex].values || [];
      const inputAlreadyExists = existingValues.some(v => v.name === input.name);

      if (!inputAlreadyExists) {
        newMappings[existingEmptyMappingIndex].values = [
          ...existingValues,
          { name: input.name, value: '' }
        ];
        setInputMappings(newMappings);
      }
    } else {
      // Create new mapping with ALL inputs (API requirement)
      const allInputValues = inputs.map(inp => ({
        name: inp.name,
        value: inp.name === input.name ? '' : ''
      }));

      const newMapping = {
        when: { triggeredBy: { connection: '' } },
        values: allInputValues
      };
      setInputMappings([...inputMappings, newMapping]);
    }
  };

  const handleRemoveMapping = (actualMappingIndex: number) => {
    const newMappings = [...inputMappings];
    newMappings.splice(actualMappingIndex, 1);
    setInputMappings(newMappings);
  };

  const handleStaticValueChange = (
    value: string,
    actualMappingIndex: number,
    inputName: string | undefined
  ) => {
    if (!inputName) return;
    const newMappings = [...inputMappings];
    const values = [...(newMappings[actualMappingIndex].values || [])];
    const valueIndex = values.findIndex(v => v.name === inputName);

    if (valueIndex !== -1) {
      values[valueIndex] = { ...values[valueIndex], value };
      newMappings[actualMappingIndex].values = values;
      setInputMappings(newMappings);
    }
  };

  const handleLastExecutionChange = (
    result: 'RESULT_PASSED' | 'RESULT_FAILED',
    checked: boolean,
    actualMappingIndex: number,
    inputName: string | undefined
  ) => {
    if (!inputName) return;
    const newMappings = [...inputMappings];
    const values = [...(newMappings[actualMappingIndex].values || [])];
    const valueIndex = values.findIndex(v => v.name === inputName);

    if (valueIndex !== -1) {
      const currentResults = values[valueIndex]?.valueFrom?.lastExecution?.results || [];
      const newResults = checked
        ? [...currentResults, result]
        : currentResults.filter(r => r !== result);

      values[valueIndex] = {
        ...values[valueIndex],
        valueFrom: {
          lastExecution: {
            ...values[valueIndex]?.valueFrom?.lastExecution,
            results: newResults
          }
        }
      };
      newMappings[actualMappingIndex].values = values;
      setInputMappings(newMappings);
    }
  };

  const handleEventDataExpressionChange = (
    expression: string,
    actualMappingIndex: number,
    inputName: string | undefined
  ) => {
    if (!inputName) return;
    const newMappings = [...inputMappings];
    const values = [...(newMappings[actualMappingIndex].values || [])];
    const valueIndex = values.findIndex(v => v.name === inputName);

    if (valueIndex !== -1) {
      values[valueIndex] = {
        ...values[valueIndex],
        valueFrom: {
          eventData: {
            ...values[valueIndex]?.valueFrom?.eventData,
            expression
          }
        }
      };
      newMappings[actualMappingIndex].values = values;
      setInputMappings(newMappings);
    }
  };

  const handleEventDataConnectionChange = (
    connection: string,
    actualMappingIndex: number,
    inputName: string | undefined
  ) => {
    if (!inputName) return;
    const newMappings = [...inputMappings];
    const values = [...(newMappings[actualMappingIndex].values || [])];
    const valueIndex = values.findIndex(v => v.name === inputName);

    if (valueIndex !== -1) {
      values[valueIndex] = {
        ...values[valueIndex],
        valueFrom: {
          eventData: {
            ...values[valueIndex]?.valueFrom?.eventData,
            connection
          }
        }
      };
      newMappings[actualMappingIndex].values = values;
      setInputMappings(newMappings);
    }
  };

  return {
    handleAddMapping,
    handleRemoveMapping,
    handleStaticValueChange,
    handleLastExecutionChange,
    handleEventDataExpressionChange,
    handleEventDataConnectionChange
  };
};