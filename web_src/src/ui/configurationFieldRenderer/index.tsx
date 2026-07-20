import React from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import type { FieldRendererProps, ValidationError } from "./types";
import { BooleanFieldRenderer } from "./BooleanFieldRenderer";
import { isFieldVisible, isFieldRequired, parseDefaultValues, validateFieldForSubmission } from "../../lib/components";
import type { AuthorizationDomainType } from "@/api-client";
import { buildTemplateParametersAutocompleteObject } from "./templateParametersAutocomplete";
import { getRunTitlePresentation, RUN_TITLE_EXCLUDED_SUGGESTIONS } from "./runTitlePresentation";
import { ReadonlyConfigurationField } from "./ReadonlyFieldRenderer";
import { ConfigurationFieldInput } from "./ConfigurationFieldInput";
import { buildReadonlyExpressionPreview } from "./expressionPreview";

const REQUIRED_FIELD_BADGE_CLASS =
  "ml-2 inline-flex items-center rounded border border-orange-300 px-1 py-0.5 text-[10px] uppercase tracking-wide leading-none text-orange-500 bg-orange-50 dark:border-orange-400/50 dark:bg-orange-950/30 dark:text-orange-300";

interface ConfigurationFieldRendererProps extends FieldRendererProps {
  allowExpressions?: boolean;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  integrationId?: string;
  organizationId?: string;
  hasError?: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
  // New real-time validation props
  realtimeValidationErrors?: Array<{ field: string; message: string; type: string }>;
  enableRealtimeValidation?: boolean;
}

type ConfigurationField = FieldRendererProps["field"];

function getInitialSelectValue(field: ConfigurationField, parsedDefaultValue: unknown): unknown {
  const selectOptions = field.typeOptions?.select?.options;
  if (!selectOptions) {
    return "";
  }

  if (parsedDefaultValue && selectOptions.some((opt) => opt.value === parsedDefaultValue)) {
    return parsedDefaultValue;
  }

  return selectOptions.length > 0 ? selectOptions[0].value : "";
}

function getInitialListValue(parsedDefaultValue: unknown): unknown[] {
  return Array.isArray(parsedDefaultValue) ? parsedDefaultValue : [];
}

function isArrayBackedTogglableField(field: ConfigurationField): boolean {
  return (
    field.type === "list" ||
    field.type === "multi-select" ||
    field.type === "days-of-week" ||
    field.type === "any-predicate-list"
  );
}

function getInitialObjectValue(parsedDefaultValue: unknown): Record<string, unknown> {
  if (parsedDefaultValue && typeof parsedDefaultValue === "object" && !Array.isArray(parsedDefaultValue)) {
    return parsedDefaultValue as Record<string, unknown>;
  }

  return {};
}

function getInitialPrimitiveToggleValue(field: ConfigurationField, parsedDefaultValue: unknown): unknown | undefined {
  if (field.type === "object") {
    return getInitialObjectValue(parsedDefaultValue);
  }

  if (field.type === "number") {
    return typeof parsedDefaultValue === "number" ? parsedDefaultValue : 0;
  }

  if (field.type === "boolean") {
    return typeof parsedDefaultValue === "boolean" ? parsedDefaultValue : false;
  }

  return undefined;
}

function getInitialTogglableValue(field: ConfigurationField, parsedDefaultValue: unknown): unknown {
  if (field.type === "select") {
    return getInitialSelectValue(field, parsedDefaultValue);
  }

  if (isArrayBackedTogglableField(field)) {
    return getInitialListValue(parsedDefaultValue);
  }

  if (field.type === "integration-resource") {
    return field.typeOptions?.resource?.multi ? getInitialListValue(parsedDefaultValue) : (parsedDefaultValue ?? "");
  }

  const primitiveValue = getInitialPrimitiveToggleValue(field, parsedDefaultValue);
  if (primitiveValue !== undefined) {
    return primitiveValue;
  }

  return parsedDefaultValue ?? "";
}

export const ConfigurationFieldRenderer = ({
  field,
  value,
  onChange,
  allValues = {},
  domainId,
  domainType,
  integrationId,
  organizationId,
  hasError = false,
  validationErrors,
  fieldPath,
  realtimeValidationErrors,
  enableRealtimeValidation = false,
  autocompleteExampleObj,
  allowExpressions = false,
  readOnly = false,
  expressionPreviewContext,
  expressionErrorMessage,
  expressionTemplateValue,
}: ConfigurationFieldRendererProps) => {
  const isTogglable = field.togglable === true;
  const isEnabled = isTogglable ? value !== null && value !== undefined : true;
  const labelRightRef = React.useRef<HTMLDivElement | null>(null);
  const [labelRightReady, setLabelRightReady] = React.useState(false);
  React.useLayoutEffect(() => {
    setLabelRightReady(true);
  }, []);

  const parsedDefaultValue = React.useMemo(() => {
    if (!field.name) return undefined;
    return parseDefaultValues([field])[field.name];
  }, [field]);

  const handleToggleChange = React.useCallback(
    (checked: boolean) => {
      if (!isTogglable) return;

      onChange(checked ? getInitialTogglableValue(field, parsedDefaultValue) : null);
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

    const errors = validateFieldForSubmission(field, value);
    return errors.map((error) => ({
      field: field.name!,
      message: error,
      type: "validation_rule" as const,
    }));
  }, [field, value, validationErrors, enableRealtimeValidation]);

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
  }, [allFieldErrors, isRequired, value, validationErrors, enableRealtimeValidation, hasError]);

  const resolvedAutocompleteExampleObj = React.useMemo(() => {
    if (field.name !== "payload") {
      return autocompleteExampleObj;
    }

    const parameters = buildTemplateParametersAutocompleteObject(allValues);
    if (!parameters) {
      return autocompleteExampleObj;
    }

    return {
      ...(autocompleteExampleObj ?? {}),
      parameters,
    };
  }, [field.name, allValues, autocompleteExampleObj]);

  if (!isVisible) {
    return null;
  }

  const fieldAllowsExpressions =
    allowExpressions && !(field.type === "string" && field.typeOptions?.string?.allowExpressions === false);
  const runTitlePresentation = getRunTitlePresentation(field.name, isEnabled);
  // `field.label` arrives as an empty string (not undefined) when a component omits it,
  // so fall back to the field name whenever the label is blank.
  const fieldLabel = runTitlePresentation?.label || field.label || field.name;
  const fieldDescription = runTitlePresentation?.description ?? field.description;

  const commonProps = {
    field,
    value,
    onChange,
    allValues,
    hasError: hasFieldError,
    autocompleteExampleObj: resolvedAutocompleteExampleObj,
    integrationId,
    organizationId,
    allowExpressions: fieldAllowsExpressions,
    readOnly,
    excludedSuggestions: runTitlePresentation ? RUN_TITLE_EXCLUDED_SUGGESTIONS : undefined,
    valuePreviewLabel: runTitlePresentation?.previewLabel,
    expressionPreviewContext,
    expressionErrorMessage,
    expressionTemplateValue,
  };

  if (readOnly && !shouldRenderFieldForReadOnly(field)) {
    const expressionPreview = buildReadonlyExpressionPreview({
      field,
      value,
      templateValue: expressionTemplateValue,
      context: expressionPreviewContext,
      errorMessage: expressionErrorMessage,
    });

    return (
      <ReadonlyConfigurationField
        field={field}
        label={fieldLabel}
        description={fieldDescription}
        value={value}
        isTogglable={isTogglable}
        isEnabled={isEnabled}
        expressionPreview={expressionPreview}
      />
    );
  }

  const renderField = () => (
    <ConfigurationFieldInput
      commonProps={commonProps}
      domainId={domainId}
      domainType={domainType}
      integrationId={integrationId}
      organizationId={organizationId}
      allowExpressions={allowExpressions}
      autocompleteExampleObj={autocompleteExampleObj}
      isRequired={isRequired}
      validationErrors={validationErrors}
      fieldPath={fieldPath}
      labelRightRef={labelRightRef}
      labelRightReady={labelRightReady}
    />
  );

  // Togglable booleans use the standard label row plus an optional labeled value switch.
  if (field.type === "boolean" && isTogglable) {
    return (
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <Switch checked={isEnabled} onCheckedChange={handleToggleChange} />
          <Label className="block text-left flex-1 min-w-0">
            {fieldLabel}
            {isRequired && <span className="text-gray-800 dark:text-gray-100 ml-1">*</span>}
            {hasFieldError &&
              ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
                (!enableRealtimeValidation &&
                  validationErrors &&
                  isRequired &&
                  (value === undefined || value === null || value === ""))) && (
                <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span>
              )}
          </Label>
        </div>
        {isEnabled && (
          <div className="flex items-center gap-3">
            <BooleanFieldRenderer {...commonProps} labeled />
          </div>
        )}

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

        {fieldDescription && (
          <p className="text-xs text-gray-500 dark:text-gray-400 text-left leading-normal">{fieldDescription}</p>
        )}
      </div>
    );
  }

  // Non-togglable booleans render the switch inline with the label.
  if (field.type === "boolean") {
    return (
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          {renderField()}
          <Label className="text-left cursor-pointer">
            {fieldLabel}
            {isRequired && <span className="text-gray-800 dark:text-gray-100 ml-1">*</span>}
            {hasFieldError &&
              ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
                (!enableRealtimeValidation &&
                  validationErrors &&
                  isRequired &&
                  (value === undefined || value === null || value === ""))) && (
                <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span>
              )}
          </Label>
        </div>

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

        {fieldDescription && (
          <p className="text-xs text-gray-500 dark:text-gray-400 text-left leading-normal">{fieldDescription}</p>
        )}
      </div>
    );
  }

  // For all other field types, render label above field
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        {isTogglable && <Switch checked={isEnabled} onCheckedChange={handleToggleChange} />}
        <Label className="block text-left flex-1 min-w-0">
          {fieldLabel}
          {isRequired && <span className="text-gray-800 dark:text-gray-100 ml-1">*</span>}
          {hasFieldError &&
            ((enableRealtimeValidation && isRequired && (value === undefined || value === null || value === "")) ||
              (!enableRealtimeValidation &&
                validationErrors &&
                isRequired &&
                (value === undefined || value === null || value === ""))) && (
              <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span>
            )}
        </Label>
        <div ref={labelRightRef} className="ml-auto shrink-0" />
      </div>
      {isEnabled && (
        <div className="flex items-center gap-2">
          <div className="flex-1 min-w-0">{renderField()}</div>
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
      {fieldDescription && (
        <p className="text-xs text-gray-500 dark:text-gray-400 text-left leading-normal">{fieldDescription}</p>
      )}
    </div>
  );
};

function shouldRenderFieldForReadOnly(field: ConfigurationField): boolean {
  return (
    field.type === "list" ||
    field.type === "select" ||
    field.type === "multi-select" ||
    field.type === "user" ||
    field.type === "role" ||
    field.type === "group" ||
    field.type === "app" ||
    (field.type === "object" && Boolean(field.typeOptions?.object?.schema?.length))
  );
}
