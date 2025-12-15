import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ConfigurationField } from "@/api-client";
import { isFieldRequired, isFieldVisible, validateFieldForSubmission } from "@/utils/components";

interface ValidationError {
  field: string;
  message: string;
  type: "validation_rule" | "required" | "visibility";
}

interface UseRealtimeValidationOptions {
  debounceMs?: number;
  validateOnMount?: boolean;
}

export function useRealtimeValidation(
  fields: ConfigurationField[],
  values: Record<string, unknown>,
  options: UseRealtimeValidationOptions = {},
) {
  const { debounceMs = 300, validateOnMount = false } = options;

  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([]);
  const [isValidating, setIsValidating] = useState(false);
  const debounceTimeoutRef = useRef<NodeJS.Timeout | undefined>(undefined);
  const lastValidationRef = useRef<string>("");

  // Memoized validation function to prevent unnecessary re-runs
  const validateAllFields = useCallback(() => {
    const errors: ValidationError[] = [];

    const validateNestedFields = (
      fieldsToValidate: ConfigurationField[],
      currentValues: Record<string, unknown>,
      fieldPath = "",
    ) => {
      fieldsToValidate.forEach((field) => {
        if (!field.name) return;

        const fullFieldName = fieldPath ? `${fieldPath}.${field.name}` : field.name;
        const value = currentValues[field.name];

        // Check if field is visible - skip validation for invisible fields
        if (!isFieldVisible(field, currentValues)) {
          return;
        }

        // Check required validation
        const isRequired = isFieldRequired(field, currentValues);
        if (isRequired && (value === undefined || value === null || value === "")) {
          errors.push({
            field: fullFieldName,
            message: "This field is required",
            type: "required",
          });
        }

        // Run field-specific validation only if field has a value
        if (value !== undefined && value !== null && value !== "") {
          // Skip expensive validation for clearly invalid input to improve performance
          let shouldValidate = true;

          if (field.type === "cron") {
            const cronValue = String(value).trim();
            // Skip validation if it's clearly incomplete (less than minimum viable cron)
            if (cronValue.length < 5 || cronValue.split(/\s+/).length < 3) {
              shouldValidate = false;
            }
          }

          if (shouldValidate) {
            const fieldErrors = validateFieldForSubmission(field, value, currentValues);
            fieldErrors.forEach((message) => {
              errors.push({
                field: fullFieldName,
                message,
                type: "validation_rule",
              });
            });
          }
        }

        // Handle nested object validation
        if (field.type === "object" && field.typeOptions?.object?.schema && value) {
          validateNestedFields(field.typeOptions.object.schema, value as Record<string, unknown>, fullFieldName);
        }

        // Handle list validation (if list has item validation schema)
        if (field.type === "list" && Array.isArray(value)) {
          // For now, we'll skip list item validation since the API doesn't have itemSchema
          // This can be extended in the future if itemSchema is added to the list type options
        }
      });
    };

    validateNestedFields(fields, values);
    return errors;
  }, [fields, values]);

  // Debounced validation function
  const debouncedValidate = useCallback(() => {
    // Create a lightweight hash to avoid duplicate validations
    const fieldNames = fields.map((f) => f.name).join(",");
    const valuesHash = JSON.stringify(values);
    const currentHash = `${fieldNames}:${valuesHash}`;

    if (currentHash === lastValidationRef.current) {
      return;
    }

    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current);
    }

    // Only set validating state if we're going to actually validate
    setIsValidating(true);

    debounceTimeoutRef.current = setTimeout(() => {
      // Double-check the hash hasn't changed during the timeout
      const recheckHash = `${fields.map((f) => f.name).join(",")}:${JSON.stringify(values)}`;
      if (recheckHash !== lastValidationRef.current) {
        const errors = validateAllFields();
        setValidationErrors(errors);
        lastValidationRef.current = recheckHash;
      }
      setIsValidating(false);
    }, debounceMs);
  }, [validateAllFields, debounceMs, fields, values]);

  // Trigger validation when values change
  useEffect(() => {
    debouncedValidate();

    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }
    };
  }, [debouncedValidate]);

  // Optional validation on mount
  useEffect(() => {
    if (validateOnMount) {
      const errors = validateAllFields();
      setValidationErrors(errors);
    }
  }, [validateOnMount, validateAllFields]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }
    };
  }, []);

  // Helper function to get errors for a specific field
  const getFieldErrors = useCallback(
    (fieldName: string) => {
      return validationErrors.filter(
        (error) =>
          error.field === fieldName ||
          error.field.startsWith(`${fieldName}.`) ||
          error.field.startsWith(`${fieldName}[`),
      );
    },
    [validationErrors],
  );

  // Helper function to check if a field has errors
  const hasFieldError = useCallback(
    (fieldName: string) => {
      return getFieldErrors(fieldName).length > 0;
    },
    [getFieldErrors],
  );

  // Helper function to check if form is valid
  const isValid = useMemo(() => {
    return validationErrors.length === 0 && !isValidating;
  }, [validationErrors, isValidating]);

  // Helper function to trigger immediate validation (for submit)
  const validateNow = useCallback((): boolean => {
    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current);
    }

    const errors = validateAllFields();
    setValidationErrors(errors);
    setIsValidating(false);
    return errors.length === 0;
  }, [validateAllFields]);

  return {
    validationErrors,
    isValidating,
    isValid,
    getFieldErrors,
    hasFieldError,
    validateNow,
    clearErrors: () => setValidationErrors([]),
  };
}
