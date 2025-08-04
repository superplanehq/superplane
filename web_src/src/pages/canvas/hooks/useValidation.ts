import { useCallback } from 'react';

export interface ValidationRule<T> {
  validate: (item: T, index?: number, allItems?: T[]) => string[];
}

export function useValidation<T>() {
  const validateName = useCallback((name: string | undefined, allItems: T[], currentIndex: number, nameField: keyof T = 'name' as keyof T): string[] => {
    const errors: string[] = [];
    
    if (!name || name.trim() === '') {
      errors.push('Name is required');
    }
    
    if (name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(name)) {
      errors.push('Name must start with a letter and contain only letters, numbers, and underscores');
    }
    
    const duplicateIndex = allItems.findIndex((item, i) => i !== currentIndex && item[nameField] === name);
    if (duplicateIndex !== -1) {
      errors.push('Name must be unique');
    }
    
    return errors;
  }, []);

  const validateRequired = useCallback((value: unknown, fieldName: string): string[] => {
    const errors: string[] = [];
    
    if (value === undefined || value === null || (typeof value === 'string' && value.trim() === '')) {
      errors.push(`${fieldName} is required`);
    }
    
    return errors;
  }, []);

  const validateUrl = useCallback((url: string | undefined): string[] => {
    const errors: string[] = [];
    
    if (url && !/^https?:\/\/.+/.test(url)) {
      errors.push('URL must be a valid HTTP/HTTPS URL');
    }
    
    return errors;
  }, []);

  const validatePositiveNumber = useCallback((value: number | undefined, fieldName: string): string[] => {
    const errors: string[] = [];
    
    if (value !== undefined && value < 0) {
      errors.push(`${fieldName} must be a positive number`);
    }
    
    return errors;
  }, []);

  const validateMinValue = useCallback((value: number | undefined, min: number, fieldName: string): string[] => {
    const errors: string[] = [];
    
    if (value !== undefined && value < min) {
      errors.push(`${fieldName} must be at least ${min}`);
    }
    
    return errors;
  }, []);

  const combineValidationResults = useCallback((validationResults: string[][]): string[] => {
    return validationResults.flat();
  }, []);

  return {
    validateName,
    validateRequired,
    validateUrl,
    validatePositiveNumber,
    validateMinValue,
    combineValidationResults
  };
}