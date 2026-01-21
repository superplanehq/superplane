import React from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";
import { StringFieldRenderer } from "./StringFieldRenderer";
import { ExpressionFieldRenderer } from "./ExpressionFieldRenderer";
import { TextFieldRenderer } from "./TextFieldRenderer";
import { XMLFieldRenderer } from "./XMLFieldRenderer";
import { NumberFieldRenderer } from "./NumberFieldRenderer";
import { BooleanFieldRenderer } from "./BooleanFieldRenderer";
import { SelectFieldRenderer } from "./SelectFieldRenderer";
import { MultiSelectFieldRenderer } from "./MultiSelectFieldRenderer";
import { DateFieldRenderer } from "./DateFieldRenderer";
import { DateTimeFieldRenderer } from "./DateTimeFieldRenderer";
import { UrlFieldRenderer } from "./UrlFieldRenderer";
import { ListFieldRenderer } from "./ListFieldRenderer";
import { ObjectFieldRenderer } from "./ObjectFieldRenderer";
import { AppInstallationResourceFieldRenderer } from "./AppInstallationResourceFieldRenderer";
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
  appInstallationId?: string;
  organizationId?: string;
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
  appInstallationId,
  organizationId,
  hasError = false,
  validationErrors,
  fieldPath,
  realtimeValidationErrors,
  enableRealtimeValidation = false,
  autocompleteExampleObj,
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
        } else if (
          field.type === "list" ||
          field.type === "multi-select" ||
          field.type === "any-predicate-list" ||
          (field.type === "app-installation-resource" && field.typeOptions?.resource?.multi)
        ) {
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
  }, [allFieldErrors, isRequired, value, validationErrors, enableRealtimeValidation]);

  if (!isVisible) {
    return null;
  }
  const renderField = () => {
    const commonProps = {
      field,
      value,
      onChange,
      allValues,
      hasError: hasFieldError,
      autocompleteExampleObj,
      appInstallationId,
      organizationId,
    };

    switch (field.type) {
      case "string":
        return <StringFieldRenderer {...commonProps} />;

      case "expression":
        return <ExpressionFieldRenderer {...commonProps} />;

      case "text":
        return <TextFieldRenderer {...commonProps} />;

      case "xml":
        return <XMLFieldRenderer {...commonProps} />;

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

      case "app-installation-resource":
        return (
          <AppInstallationResourceFieldRenderer
            field={field}
            value={value as string | string[] | undefined}
            onChange={onChange}
            organizationId={organizationId}
            appInstallationId={appInstallationId}
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
            {...commonProps}
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
          {isTogglable && <Switch checked={isEnabled} onCheckedChange={handleToggleChange} />}
          {isEnabled && renderField()}
          <Label className="text-left cursor-pointer">
            {field.label || field.name}
            {isRequired && <span className="text-gray-800 ml-1">*</span>}
            {hasFieldError &&
              ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
                (!enableRealtimeValidation &&
                  validationErrors &&
                  isRequired &&
                  (value === undefined || value === null || value === ""))) && (
                <span className="text-red-500 text-xs ml-2">Required</span>
              )}
          </Label>
        </div>

        {/* Display validation errors */}
        {allFieldErrors.filter((error) => error.message && !error.message.toLowerCase().includes("required")).length >
          0 && (
          <div className="space-y-1">
            {allFieldErrors
              .filter((error) => error.message && !error.message.toLowerCase().includes("required"))
              .map((error, index) => (
                <p key={index} className="text-xs text-red-500 dark:text-red-400 text-left">
                  {error.message}
                </p>
              ))}
          </div>
        )}

        {/* Display field description */}
        {field.description && (
          <p className="text-xs text-gray-500 dark:text-gray-400 text-left leading-normal">
            {field.description}
          </p>
        )}
      </div>
    );
  }

  // For all other field types, render label above field
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        {isTogglable && <Switch checked={isEnabled} onCheckedChange={handleToggleChange} />}
        <Label className="block text-left">
          {field.label || field.name}
          {isRequired && <span className="text-gray-800 ml-1">*</span>}
          {hasFieldError &&
            ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
              (!enableRealtimeValidation &&
                validationErrors &&
                isRequired &&
                (value === undefined || value === null || value === ""))) && (
              <span className="text-red-500 text-xs ml-2 leading-0">Required</span>
            )}
        </Label>
      </div>
      {isEnabled && (
        <div className="flex items-center gap-2">
          <div className="flex-1">{renderField()}</div>
        </div>
      )}

      {/* Display validation errors */}
      {allFieldErrors.filter((error) => error.message && !error.message.toLowerCase().includes("required")).length >
        0 && (
        <div className="space-y-1">
          {allFieldErrors
            .filter((error) => error.message && !error.message.toLowerCase().includes("required"))
            .map((error, index) => (
              <p key={index} className="text-xs text-red-500 dark:text-red-400 text-left">
                {error.message}
              </p>
            ))}
        </div>
      )}

      {/* Display field description */}
      {field.description && (
        <p className="text-xs text-gray-500 dark:text-gray-400 text-left leading-normal">
          {field.description}
        </p>
      )}
    </div>
  );
};
