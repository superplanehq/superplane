import React from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";
import { StringFieldRenderer } from "./StringFieldRenderer";
import { TextFieldRenderer } from "./TextFieldRenderer";
import { XMLFieldRenderer } from "./XMLFieldRenderer";
import { NumberFieldRenderer } from "./NumberFieldRenderer";
import { BooleanFieldRenderer } from "./BooleanFieldRenderer";
import { SelectFieldRenderer } from "./SelectFieldRenderer";
import { RadioButtonFieldRenderer } from "./RadioButtonFieldRenderer";
import { MultiSelectFieldRenderer } from "./MultiSelectFieldRenderer";
import { DaysOfWeekToggle } from "./DaysOfWeekToggle";
import { DateFieldRenderer } from "./DateFieldRenderer";
import { DateTimeFieldRenderer } from "./DateTimeFieldRenderer";
import { UrlFieldRenderer } from "./UrlFieldRenderer";
import { ListFieldRenderer } from "./ListFieldRenderer";
import { ObjectFieldRenderer } from "./ObjectFieldRenderer";
import { IntegrationFieldRenderer } from "./IntegrationFieldRenderer";
import { IntegrationResourceFieldRenderer } from "./IntegrationResourceFieldRenderer";
import { TimeFieldRenderer } from "./TimeFieldRenderer";
import { DayInYearFieldRenderer } from "./DayInYearFieldRenderer";
import { CronFieldRenderer } from "./CronFieldRenderer";
import { UserFieldRenderer } from "./UserFieldRenderer";
import { RoleFieldRenderer } from "./RoleFieldRenderer";
import { GroupFieldRenderer } from "./GroupFieldRenderer";
import { GitRefFieldRenderer } from "./GitRefFieldRenderer";
import { TimezoneFieldRenderer } from "./TimezoneFieldRenderer";
import { AnyPredicateListFieldRenderer } from "./AnyPredicateListFieldRenderer";
import {
  isFieldVisible,
  isFieldRequired,
  parseDefaultValues,
  validateFieldForSubmission,
} from "../../utils/components";
import { ValidationError } from "./types";
import { AuthorizationDomainType } from "@/api-client";

interface ConfigurationFieldRendererProps extends FieldRendererProps {
  domainId?: string;
  domainType?: AuthorizationDomainType;
  hasError?: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
  // New real-time validation props
  realtimeValidationErrors?: Array<{ field: string; message: string; type: string }>;
  enableRealtimeValidation?: boolean;
}

export const ConfigurationFieldRenderer = ({
  field,
  value,
  onChange,
  allValues = {},
  domainId,
  domainType,
  hasError = false,
  validationErrors,
  fieldPath,
  realtimeValidationErrors,
  enableRealtimeValidation = false,
}: ConfigurationFieldRendererProps) => {
  const isTogglable = field.togglable === true;
  const isEnabled = isTogglable ? value !== null && value !== undefined : true;

  const parsedDefaultValue = React.useMemo(() => {
    if (!field.name) return undefined;
    return parseDefaultValues([field])[field.name];
  }, [field]);

  const handleToggleChange = React.useCallback(
    (checked: boolean) => {
      if (!isTogglable) return;

      if (checked) {
        if (field.type === "select" && field.typeOptions?.select?.options) {
          const selectOptions = field.typeOptions.select.options;
          const initialValue =
            parsedDefaultValue && selectOptions.some((opt) => opt.value === parsedDefaultValue)
              ? parsedDefaultValue
              : selectOptions.length > 0
                ? selectOptions[0].value
                : "";
          onChange(initialValue);
        } else if (field.type === "list" || field.type === "multi-select" || field.type === "any-predicate-list") {
          onChange(Array.isArray(parsedDefaultValue) ? parsedDefaultValue : []);
        } else if (field.type === "object") {
          onChange(
            parsedDefaultValue && typeof parsedDefaultValue === "object" && !Array.isArray(parsedDefaultValue)
              ? parsedDefaultValue
              : {},
          );
        } else if (field.type === "number") {
          onChange(typeof parsedDefaultValue === "number" ? parsedDefaultValue : 0);
        } else if (field.type === "boolean") {
          onChange(typeof parsedDefaultValue === "boolean" ? parsedDefaultValue : false);
        } else {
          onChange(parsedDefaultValue ?? "");
        }
      } else {
        onChange(null);
      }
    },
    [isTogglable, field, onChange, parsedDefaultValue],
  );

  // Check visibility conditions
  const isVisible = React.useMemo(() => {
    return isFieldVisible(field, allValues);
  }, [field, allValues]);

  // Check if field is conditionally required
  const isRequired = React.useMemo(() => {
    return isFieldRequired(field, allValues);
  }, [field, allValues]);

  // Validate field value (only when validation is explicitly requested and real-time validation is disabled)
  const fieldValidationErrors = React.useMemo(() => {
    // Only run this validation if real-time validation is disabled
    if (!field.name || !validationErrors || enableRealtimeValidation) return [];

    const errors = validateFieldForSubmission(field, value, allValues);
    return errors.map((error) => ({
      field: field.name!,
      message: error,
      type: "validation_rule" as const,
    }));
  }, [field, value, allValues, validationErrors, enableRealtimeValidation]);

  // Get field-specific validation errors
  const fieldErrors = React.useMemo(() => {
    const errors: ValidationError[] = [];

    if (field.name && validationErrors) {
      if (validationErrors instanceof Set) {
        // Handle legacy Set<string> format
        const fieldName = field.name;
        const fieldPathName = fieldPath || fieldName;
        const hasError =
          validationErrors.has(fieldName) ||
          Array.from(validationErrors).some(
            (error) => error.startsWith(`${fieldPathName}.`) || error.startsWith(`${fieldPathName}[`),
          );

        if (hasError) {
          errors.push({
            field: fieldName,
            message: "",
            type: "validation_rule" as const,
          });
        }
      } else {
        // Handle new ValidationError[] format
        const matchingErrors = validationErrors.filter(
          (error) => error.field === field.name || error.field.startsWith(`${fieldPath || field.name}.`),
        );
        errors.push(...matchingErrors);
      }
    }

    // Add real-time validation errors if enabled
    if (field.name && enableRealtimeValidation && realtimeValidationErrors) {
      const realtimeErrors = realtimeValidationErrors
        .filter(
          (error) =>
            error.field === field.name ||
            error.field.startsWith(`${fieldPath || field.name}.`) ||
            error.field.startsWith(`${fieldPath || field.name}[`),
        )
        .map((error) => ({
          field: error.field,
          message: error.message,
          type: error.type as "validation_rule" | "required" | "visibility",
        }));
      errors.push(...(realtimeErrors as ValidationError[]));
    }

    return errors;
  }, [validationErrors, realtimeValidationErrors, enableRealtimeValidation, field.name, fieldPath]);

  // Combine all errors
  const allFieldErrors = React.useMemo(() => {
    return [...fieldErrors, ...fieldValidationErrors];
  }, [fieldErrors, fieldValidationErrors]);

  // Check if there are any errors or if required field is empty
  const hasFieldError = React.useMemo(() => {
    if (allFieldErrors.length > 0) return true;

    // For real-time validation, check if required field is empty
    if (enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) {
      return true;
    }

    // For traditional validation, check if validation is shown and required field is empty
    if (
      !enableRealtimeValidation &&
      validationErrors &&
      isRequired &&
      (value === undefined || value === null || value === "")
    ) {
      return true;
    }

    return hasError;
  }, [allFieldErrors, isRequired, value, hasError, validationErrors, enableRealtimeValidation]);

  if (!isVisible) {
    return null;
  }
  const renderField = () => {
    const commonProps = { field, value, onChange, allValues, hasError: hasFieldError };

    switch (field.type) {
      case "string":
        return <StringFieldRenderer {...commonProps} />;

      case "text":
        return <TextFieldRenderer {...commonProps} />;

      case "xml":
        return <XMLFieldRenderer {...commonProps} />;

      case "number":
        return <NumberFieldRenderer {...commonProps} />;

      case "boolean":
        return <BooleanFieldRenderer {...commonProps} />;

      case "select":
        // Use radio buttons for select fields with exactly 2 options (like Mode and Type)
        const selectOptions = field.typeOptions?.select?.options ?? [];
        if (selectOptions.length === 2) {
          return <RadioButtonFieldRenderer {...commonProps} />;
        }
        return <SelectFieldRenderer {...commonProps} />;

      case "multi-select":
        // Use special toggle component for days of week field
        if (field.name === "days") {
          return <DaysOfWeekToggle {...commonProps} />;
        }
        return <MultiSelectFieldRenderer {...commonProps} />;

      case "date":
        return <DateFieldRenderer {...commonProps} />;

      case "datetime":
        return <DateTimeFieldRenderer {...commonProps} />;

      case "url":
        return <UrlFieldRenderer {...commonProps} />;

      case "time":
        return <TimeFieldRenderer {...commonProps} />;

      case "day-in-year":
        return <DayInYearFieldRenderer {...commonProps} />;

      case "cron":
        return <CronFieldRenderer {...commonProps} />;

      case "integration":
        return (
          <IntegrationFieldRenderer
            field={field}
            value={value as string}
            onChange={onChange}
            domainId={domainId}
            domainType={domainType}
          />
        );

      case "integration-resource":
        return (
          <IntegrationResourceFieldRenderer
            field={field}
            value={value as string}
            onChange={onChange}
            allValues={allValues}
            domainId={domainId}
            domainType={domainType}
          />
        );

      case "git-ref":
        return <GitRefFieldRenderer {...commonProps} />;

      case "user":
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">User field requires domainId prop</div>;
        }
        return (
          <UserFieldRenderer
            field={field}
            value={value as string}
            onChange={onChange}
            domainId={domainId}
            allValues={allValues}
          />
        );

      case "role":
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">Role field requires domainId prop</div>;
        }
        return (
          <RoleFieldRenderer
            field={field}
            value={value as string}
            onChange={onChange}
            domainId={domainId}
            allValues={allValues}
          />
        );

      case "group":
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">Group field requires domainId prop</div>;
        }
        return (
          <GroupFieldRenderer
            field={field}
            value={value as string}
            onChange={onChange}
            domainId={domainId}
            allValues={allValues}
          />
        );

      case "list":
        return (
          <ListFieldRenderer
            {...commonProps}
            domainId={domainId}
            domainType={domainType}
            validationErrors={validationErrors}
            fieldPath={fieldPath || field.name}
          />
        );

      case "any-predicate-list":
        return <AnyPredicateListFieldRenderer {...commonProps} />;

      case "object":
        return <ObjectFieldRenderer {...commonProps} domainId={domainId} domainType={domainType} />;

      case "timezone":
        return <TimezoneFieldRenderer {...commonProps} />;

      default:
        // Fallback to text input
        return <StringFieldRenderer {...commonProps} />;
    }
  };

  // For boolean fields, render label inline with switch
  if (field.type === "boolean") {
    return (
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          {isTogglable && (
            <Switch
              checked={isEnabled}
              onCheckedChange={handleToggleChange}
              className={`${hasFieldError ? "border-red-500 border-2" : ""}`}
            />
          )}
          {isEnabled && renderField()}
          <Label className={`text-left cursor-pointer ${hasFieldError ? "text-red-600 dark:text-red-400" : ""}`}>
            {field.label || field.name}
            {isRequired && <span className="text-red-500 ml-1">*</span>}
            {hasFieldError &&
              ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
                (!enableRealtimeValidation &&
                  validationErrors &&
                  isRequired &&
                  (value === undefined || value === null || value === ""))) && (
                <span className="text-red-500 text-xs ml-2">- required field</span>
              )}
          </Label>
        </div>

        {/* Display validation errors */}
        {allFieldErrors.length > 0 && (
          <div className="space-y-1">
            {allFieldErrors.map((error, index) => (
              <p key={index} className="text-xs text-red-500 dark:text-red-400 text-left">
                {error.message}
              </p>
            ))}
          </div>
        )}

        {/* Display field description */}
        {field.description && (
          <p className="text-xs text-gray-500 dark:text-gray-400 text-left bg-gray-50 dark:bg-gray-800 p-2 rounded">
            {field.description}
          </p>
        )}
      </div>
    );
  }

  // Check if this field uses tabs (select with 2 options)
  const usesTabs = field.type === "select" && field.typeOptions?.select?.options?.length === 2;
  
  // Check if this is the items field (Time Windows) - hide label for it
  const isItemsField = field.name === "items";
  
  // Check if this is the days field - hide label for it
  const isDaysField = field.name === "days";
  
  // Check if this is a time field that's handled by TimeRangeWithAllDay - hide label for it
  const isTimeFieldInRange = field.name === "startTime" || field.name === "endTime";
  
  // Check if this is the date field for specific dates - hide label for it (will be shown in custom renderer)
  const isDateFieldInList = field.name === "date";
  
  // For all other field types, render label above field
  return (
    <div className="space-y-2">
      {!usesTabs && !isItemsField && !isDaysField && !isTimeFieldInRange && !isDateFieldInList && (
        <div className="flex items-center gap-3">
          {isTogglable && (
            <Switch
              checked={isEnabled}
              onCheckedChange={handleToggleChange}
              className={`${hasFieldError ? "border-red-500 border-2" : ""}`}
            />
          )}
          <Label className={`block text-left ${hasFieldError ? "text-red-600 dark:text-red-400" : ""}`}>
            {field.label || field.name}
            {isRequired && <span className="text-red-500 ml-1">*</span>}
            {hasFieldError &&
              ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
                (!enableRealtimeValidation &&
                  validationErrors &&
                  isRequired &&
                  (value === undefined || value === null || value === ""))) && (
                <span className="text-red-500 text-xs ml-2">- required field</span>
              )}
          </Label>
        </div>
      )}
      {isEnabled && (
        <div className="flex items-center gap-2">
          <div className="flex-1">{renderField()}</div>
        </div>
      )}

      {/* Display validation errors */}
      {allFieldErrors.length > 0 && (
        <div className="space-y-1">
          {allFieldErrors.map((error, index) => (
            <p key={index} className="text-xs text-red-500 dark:text-red-400 text-left">
              {error.message}
            </p>
          ))}
        </div>
      )}

      {/* Display field description */}
      {field.description && (
        <p className="text-xs text-gray-500 dark:text-gray-400 text-left bg-gray-50 dark:bg-gray-800 p-2 rounded">
          {field.description}
        </p>
      )}

      {/* Display type-specific help text */}
      {field.typeOptions?.number?.min !== undefined && field.typeOptions?.number?.max !== undefined && (
        <p className="text-xs text-gray-500 dark:text-gray-400 text-left">
          Range: {field.typeOptions.number.min} - {field.typeOptions.number.max}
        </p>
      )}
    </div>
  );
};
