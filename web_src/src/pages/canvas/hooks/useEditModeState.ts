import { useState, useCallback } from 'react';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
interface UseEditModeStateProps<T> {
  initialData: T;
  onDataChange?: ((data: any) => void) | undefined;
  validateAllFields: () => boolean;
}

export function useEditModeState<T extends Record<string, unknown>>({
  initialData,
  onDataChange,
  validateAllFields
}: UseEditModeStateProps<T>) {
  const [openSections, setOpenSections] = useState<string[]>(['general']);
  const [originalData] = useState(initialData);
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});
  const [isInternalUpdate, setIsInternalUpdate] = useState(false);

  const handleAccordionToggle = useCallback((sectionId: string) => {
    setOpenSections(prev => {
      return prev.includes(sectionId)
        ? prev.filter(id => id !== sectionId)
        : [...prev, sectionId];
    });
  }, []);

  const isSectionModified = useCallback((currentData: unknown, originalField: keyof T): boolean => {
    return JSON.stringify(currentData) !== JSON.stringify(originalData[originalField]);
  }, [originalData]);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleDataChange = useCallback((data: any) => {
    if (onDataChange) {
      setIsInternalUpdate(true);
      const isValid = validateAllFields();
      onDataChange({
        ...data,
        isValid
      });
    }
  }, [onDataChange, validateAllFields]);

  const syncWithIncomingData = useCallback((incomingData: T, stateSetter: (data: T) => void) => {
    if (!isInternalUpdate) {
      stateSetter(incomingData);
    }
    setIsInternalUpdate(false);
  }, [isInternalUpdate]);

  return {
    openSections,
    setOpenSections,
    originalData,
    validationErrors,
    setValidationErrors,
    isInternalUpdate,
    setIsInternalUpdate,
    handleAccordionToggle,
    isSectionModified,
    handleDataChange,
    syncWithIncomingData
  };
}