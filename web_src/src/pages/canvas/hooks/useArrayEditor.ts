import { useState, useCallback } from 'react';

interface UseArrayEditorProps<T> {
  items: T[];
  setItems: (items: T[]) => void;
  createNewItem: () => T;
  validateItem?: (item: T, index: number) => string[];
  setValidationErrors?: (errors: Record<string, string> | ((prev: Record<string, string>) => Record<string, string>)) => void;
  errorPrefix: string;
}

export function useArrayEditor<T extends Record<string, unknown>>({
  items,
  setItems,
  createNewItem,
  validateItem,
  setValidationErrors,
  errorPrefix
}: UseArrayEditorProps<T>) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const addItem = useCallback(() => {
    const newItem = createNewItem();
    const newIndex = items.length;
    setItems([...items, newItem]);
    setEditingIndex(newIndex);
  }, [items, setItems, createNewItem]);

  const updateItem = useCallback((index: number, field: keyof T, value: T[keyof T]) => {
    setItems(items.map((item, i) =>
      i === index ? { ...item, [field]: value } : item
    ));
    
    if (validateItem && setValidationErrors) {
      setTimeout(() => {
        const item = { ...items[index], [field]: value };
        const errors = validateItem(item, index);
        if (errors.length > 0) {
          setValidationErrors(prev => ({
            ...prev,
            [`${errorPrefix}_${index}`]: errors.join(', ')
          }));
        } else {
          setValidationErrors(prev => {
            const newErrors = { ...prev };
            delete newErrors[`${errorPrefix}_${index}`];
            return newErrors;
          });
        }
      }, 0);
    }
  }, [items, setItems, validateItem, setValidationErrors, errorPrefix]);

  const removeItem = useCallback((index: number) => {
    setItems(items.filter((_, i) => i !== index));
    setEditingIndex(null);
  }, [items, setItems]);

  const cancelEdit = useCallback((index: number, shouldRemoveEmpty?: (item: T) => boolean) => {
    const item = items[index];
    
    if (shouldRemoveEmpty && shouldRemoveEmpty(item)) {
      removeItem(index);
    } else {
      setEditingIndex(null);
    }
  }, [items, removeItem]);

  const saveEdit = useCallback(() => {
    setEditingIndex(null);
  }, []);

  const startEdit = useCallback((index: number) => {
    setEditingIndex(index);
  }, []);

  return {
    editingIndex,
    setEditingIndex,
    addItem,
    updateItem,
    removeItem,
    cancelEdit,
    saveEdit,
    startEdit
  };
}