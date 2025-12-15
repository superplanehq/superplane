import React from "react";
import { Label } from "../label";
import { FieldRendererProps } from "./types";
import { StringFieldRenderer } from "./StringFieldRenderer";
import { NumberFieldRenderer } from "./NumberFieldRenderer";
import { BooleanFieldRenderer } from "./BooleanFieldRenderer";
import { SelectFieldRenderer } from "./SelectFieldRenderer";
import { MultiSelectFieldRenderer } from "./MultiSelectFieldRenderer";
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
import { isFieldVisible, isFieldRequired, validateFieldForSubmission } from "../../utils/components";
import { ValidationError } from "./types";
import { AuthorizationDomainType } from "@/api-client";

interface ConfigurationFieldRendererProps extends FieldRendererProps {
  domainId?: string;
  domainType?: AuthorizationDomainType;
  hasError?: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
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
}: ConfigurationFieldRendererProps) => {
  // Check visibility conditions
  const isVisible = React.useMemo(() => {
    return isFieldVisible(field, allValues);
  }, [field, allValues]);

  // Check if field is conditionally required
  const isRequired = React.useMemo(() => {
    return isFieldRequired(field, allValues);
  }, [field, allValues]);

  // Validate field value (only when validation is explicitly requested)
  const fieldValidationErrors = React.useMemo(() => {
    if (!field.name || !validationErrors) return [];

    const errors = validateFieldForSubmission(field, value, allValues);
    return errors.map((error) => ({
      field: field.name!,
      message: error,
      type: "validation_rule" as const,
    }));
  }, [field, value, allValues, validationErrors]);

  // Get field-specific validation errors
  const fieldErrors = React.useMemo(() => {
    if (!field.name || !validationErrors) return [];

    if (validationErrors instanceof Set) {
      // Handle legacy Set<string> format
      const fieldName = field.name;
      const fieldPathName = fieldPath || fieldName;
      const hasError =
        validationErrors.has(fieldName) ||
        Array.from(validationErrors).some(
          (error) => error.startsWith(`${fieldPathName}.`) || error.startsWith(`${fieldPathName}[`),
        );

      return hasError
        ? [
            {
              field: fieldName,
              message: "",
              type: "validation_rule" as const,
            },
          ]
        : [];
    } else {
      // Handle new ValidationError[] format
      return validationErrors.filter(
        (error) => error.field === field.name || error.field.startsWith(`${fieldPath || field.name}.`),
      );
    }
  }, [validationErrors, field.name, fieldPath]);

  // Combine all errors
  const allFieldErrors = React.useMemo(() => {
    return [...fieldErrors, ...fieldValidationErrors];
  }, [fieldErrors, fieldValidationErrors]);

  // Check if there are any errors or if required field is empty (only when validation is requested)
  const hasFieldError = React.useMemo(() => {
    if (allFieldErrors.length > 0) return true;
    if (validationErrors && isRequired && (value === undefined || value === null || value === "")) return true;
    return hasError;
  }, [allFieldErrors, isRequired, value, hasError, validationErrors]);

  if (!isVisible) {
    return null;
  }
  const renderField = () => {
    const commonProps = { field, value, onChange, allValues, hasError: hasFieldError };

    switch (field.type) {
      case "string":
        return <StringFieldRenderer {...commonProps} />;

      case "number":
        return <NumberFieldRenderer {...commonProps} />;

      case "boolean":
        return <BooleanFieldRenderer {...commonProps} />;

      case "select":
        return <SelectFieldRenderer {...commonProps} />;

      case "multi-select":
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
        return <UserFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} />;

      case "role":
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">Role field requires domainId prop</div>;
        }
        return <RoleFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} />;

      case "group":
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">Group field requires domainId prop</div>;
        }
        return <GroupFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} />;

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

      case "object":
        return <ObjectFieldRenderer {...commonProps} domainId={domainId} domainType={domainType} />;

      case "timezone":
        return <TimezoneFieldRenderer {...commonProps} />;

      default:
        // Fallback to text input
        return <StringFieldRenderer {...commonProps} />;
    }
  };

  return (
    <div className="space-y-2">
      <Label className={`block text-left ${hasFieldError ? "text-red-600 dark:text-red-400" : ""}`}>
        {field.label || field.name}
        {isRequired && <span className="text-red-500 ml-1">*</span>}
        {hasFieldError && validationErrors && isRequired && (value === undefined || value === null || value === "") && (
          <span className="text-red-500 text-xs ml-2">- required field</span>
        )}
      </Label>
      <div className="flex items-center gap-2">
        <div className="flex-1">{renderField()}</div>
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

      {/* Display type-specific help text */}
      {field.typeOptions?.number?.min !== undefined && field.typeOptions?.number?.max !== undefined && (
        <p className="text-xs text-gray-500 dark:text-zinc-400 text-left">
          Range: {field.typeOptions.number.min} - {field.typeOptions.number.max}
        </p>
      )}
    </div>
  );
};
