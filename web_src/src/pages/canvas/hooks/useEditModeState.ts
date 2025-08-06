import { useState, useCallback, useRef } from 'react';

interface UseEditModeStateProps<T> {
  initialData: T;
  onDataChange?: ((data: T) => void) | undefined;
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
  const isInternalUpdateRef = useRef(false);

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
      isInternalUpdateRef.current = true;
      const isValid = validateAllFields();
      onDataChange({
        ...data,
        isValid
      });
    }
  }, [onDataChange, validateAllFields]);

  const syncWithIncomingData = useCallback((incomingData: T, stateSetter: (data: T) => void) => {
    if (!isInternalUpdateRef.current) {
      stateSetter(incomingData);
    }
    isInternalUpdateRef.current = false;
  }, []);

  return {
    openSections,
    setOpenSections,
    originalData,
    validationErrors,
    setValidationErrors,
    handleAccordionToggle,
    isSectionModified,
    handleDataChange,
    syncWithIncomingData
  };
}