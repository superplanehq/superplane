import type React from "react";
import { AnyPredicateListFieldRenderer } from "./AnyPredicateListFieldRenderer";
import { BooleanFieldRenderer } from "./BooleanFieldRenderer";
import { CronFieldRenderer } from "./CronFieldRenderer";
import { DateFieldRenderer } from "./DateFieldRenderer";
import { DateTimeFieldRenderer } from "./DateTimeFieldRenderer";
import { DayInYearFieldRenderer } from "./DayInYearFieldRenderer";
import { DaysOfWeekFieldRenderer } from "./DaysOfWeekFieldRenderer";
import { ExpressionFieldRenderer } from "./ExpressionFieldRenderer";
import { GitRefFieldRenderer } from "./GitRefFieldRenderer";
import { GroupFieldRenderer } from "./GroupFieldRenderer";
import { IntegrationResourceFieldRenderer } from "./IntegrationResourceFieldRenderer";
import { AppFieldRenderer } from "./AppFieldRenderer";
import { AppCanvasNodeFieldRenderer } from "./AppCanvasNodeFieldRenderer";
import { RunParametersFieldRenderer } from "./RunParametersFieldRenderer";
import { ListFieldRenderer } from "./ListFieldRenderer";
import { MultiSelectFieldRenderer } from "./MultiSelectFieldRenderer";
import { NumberFieldRenderer } from "./NumberFieldRenderer";
import { ObjectFieldRenderer } from "./ObjectFieldRenderer";
import { RepositoryFileFieldRenderer } from "./RepositoryFileFieldRenderer";
import { RoleFieldRenderer } from "./RoleFieldRenderer";
import { SecretKeyFieldRenderer, type SecretKeyRefValue } from "./SecretKeyFieldRenderer";
import { SelectFieldRenderer } from "./SelectFieldRenderer";
import { StringFieldRenderer } from "./StringFieldRenderer";
import { TextFieldRenderer } from "./TextFieldRenderer";
import { TimeFieldRenderer } from "./TimeFieldRenderer";
import { TimeRangeFieldRenderer } from "./TimeRangeFieldRenderer";
import { TimezoneFieldRenderer } from "./TimezoneFieldRenderer";
import type { FieldRendererProps, ValidationError } from "./types";
import { UrlFieldRenderer } from "./UrlFieldRenderer";
import { UserFieldRenderer } from "./UserFieldRenderer";
import { XMLFieldRenderer } from "./XMLFieldRenderer";

type ConfigurationFieldInputProps = {
  commonProps: FieldRendererProps;
  integrationId?: string;
  organizationId?: string;
  allowExpressions: boolean;
  autocompleteExampleObj?: Record<string, unknown> | null;
  isRequired: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
  labelRightRef: React.RefObject<HTMLDivElement | null>;
  labelRightReady: boolean;
};

function isPrincipalField(fieldType: string | undefined): boolean {
  return fieldType === "user" || fieldType === "role" || fieldType === "group";
}

function fieldRequiresOrganizationContext(fieldType: string | undefined): boolean {
  return (
    fieldType === "user" ||
    fieldType === "role" ||
    fieldType === "group" ||
    fieldType === "integration-resource" ||
    fieldType === "app" ||
    fieldType === "app-canvas-node" ||
    fieldType === "run-parameters" ||
    fieldType === "secret-key"
  );
}

function OrganizationContextRequiredMessage() {
  return <div className="text-sm text-red-500 dark:text-red-400">This field requires organization context.</div>;
}

export function ConfigurationFieldInput({
  commonProps,
  integrationId,
  organizationId,
  allowExpressions,
  autocompleteExampleObj,
  isRequired,
  validationErrors,
  fieldPath,
  labelRightRef,
  labelRightReady,
}: ConfigurationFieldInputProps) {
  const { field, value, onChange, allValues } = commonProps;

  if (fieldRequiresOrganizationContext(field.type) && !organizationId) {
    return <OrganizationContextRequiredMessage />;
  }

  if (isPrincipalField(field.type)) {
    return renderPrincipalField({ commonProps, organizationId: organizationId! });
  }

  if (field.type === "integration-resource") {
    return (
      <IntegrationResourceFieldRenderer
        field={field}
        value={value as string | string[] | undefined}
        onChange={onChange}
        allValues={allValues}
        organizationId={organizationId!}
        integrationId={integrationId}
        allowExpressions={allowExpressions}
        autocompleteExampleObj={autocompleteExampleObj}
        labelRightRef={allowExpressions ? labelRightRef : undefined}
        labelRightReady={allowExpressions ? labelRightReady : false}
      />
    );
  }

  if (field.type === "app") {
    return (
      <AppFieldRenderer
        field={field}
        value={value as string | undefined}
        onChange={onChange}
        organizationId={organizationId!}
        readOnly={commonProps.readOnly}
      />
    );
  }

  if (field.type === "app-canvas-node") {
    return (
      <AppCanvasNodeFieldRenderer
        field={field}
        value={value as string | undefined}
        onChange={onChange}
        allValues={allValues}
        organizationId={organizationId!}
        readOnly={commonProps.readOnly}
      />
    );
  }

  if (field.type === "run-parameters") {
    return (
      <RunParametersFieldRenderer
        {...commonProps}
        organizationId={organizationId!}
        allowExpressions={allowExpressions}
        autocompleteExampleObj={autocompleteExampleObj}
        validationErrors={validationErrors}
        fieldPath={fieldPath}
      />
    );
  }

  if (field.type === "secret-key") {
    return (
      <SecretKeyFieldRenderer
        field={field}
        isRequired={isRequired}
        value={value as SecretKeyRefValue}
        onChange={(nextValue) => onChange(nextValue)}
        organizationId={organizationId!}
      />
    );
  }

  if (field.type === "list") {
    return (
      <ListFieldRenderer {...commonProps} validationErrors={validationErrors} fieldPath={fieldPath || field.name} />
    );
  }

  if (field.type === "object") {
    return <ObjectFieldRenderer {...commonProps} />;
  }

  return renderStandardField({ commonProps });
}

function renderPrincipalField({
  commonProps,
  organizationId,
}: {
  commonProps: FieldRendererProps;
  organizationId: string;
}) {
  const { field, value, onChange, allValues, readOnly } = commonProps;

  if (field.type === "user") {
    return (
      <UserFieldRenderer
        field={field}
        value={value as string}
        onChange={onChange}
        organizationId={organizationId}
        allValues={allValues}
        readOnly={readOnly}
      />
    );
  }

  if (field.type === "role") {
    return (
      <RoleFieldRenderer
        field={field}
        value={value as string}
        onChange={onChange}
        organizationId={organizationId}
        allValues={allValues}
        readOnly={readOnly}
      />
    );
  }

  return (
    <GroupFieldRenderer
      {...commonProps}
      field={field}
      value={value as string}
      onChange={onChange}
      organizationId={organizationId}
      allValues={allValues}
      readOnly={readOnly}
    />
  );
}

function renderStandardField({ commonProps }: { commonProps: FieldRendererProps }) {
  const { field } = commonProps;

  if (isTextField(field.type)) {
    return renderTextField(commonProps);
  }

  if (isDateTimeField(field.type)) {
    return renderDateTimeField(commonProps);
  }

  if (isReferenceField(field.type)) {
    return renderReferenceField(commonProps);
  }

  return renderFallbackField(commonProps);
}

function renderTextField(commonProps: FieldRendererProps) {
  switch (commonProps.field.type) {
    case "string":
      return <StringFieldRenderer {...commonProps} />;
    case "expression":
      return <ExpressionFieldRenderer {...commonProps} />;
    case "text":
      return <TextFieldRenderer {...commonProps} />;
    case "xml":
      return <XMLFieldRenderer {...commonProps} />;
    default:
      return <StringFieldRenderer {...commonProps} />;
  }
}

function renderDateTimeField(commonProps: FieldRendererProps) {
  switch (commonProps.field.type) {
    case "date":
      return <DateFieldRenderer {...commonProps} />;
    case "datetime":
      return <DateTimeFieldRenderer {...commonProps} />;
    case "time":
      return <TimeFieldRenderer {...commonProps} />;
    case "time-range":
      return <TimeRangeFieldRenderer {...commonProps} />;
    case "day-in-year":
      return <DayInYearFieldRenderer {...commonProps} />;
    case "cron":
      return <CronFieldRenderer {...commonProps} />;
    default:
      return <StringFieldRenderer {...commonProps} />;
  }
}

function renderReferenceField(commonProps: FieldRendererProps) {
  switch (commonProps.field.type) {
    case "git-ref":
      return <GitRefFieldRenderer {...commonProps} />;
    case "repository-file":
      return <RepositoryFileFieldRenderer {...commonProps} />;
    case "timezone":
      return <TimezoneFieldRenderer {...commonProps} />;
    default:
      return <AnyPredicateListFieldRenderer {...commonProps} />;
  }
}

function renderFallbackField(commonProps: FieldRendererProps) {
  const { field } = commonProps;

  switch (field.type) {
    case "number":
      return <NumberFieldRenderer {...commonProps} />;
    case "boolean":
      return <BooleanFieldRenderer {...commonProps} />;
    case "select":
      return <SelectFieldRenderer {...commonProps} />;
    case "multi-select":
      return <MultiSelectFieldRenderer {...commonProps} />;
    case "days-of-week":
      return <DaysOfWeekFieldRenderer {...commonProps} />;
    case "url":
      return <UrlFieldRenderer {...commonProps} />;
    default:
      return <StringFieldRenderer {...commonProps} />;
  }
}

function isTextField(fieldType: string | undefined): boolean {
  return fieldType === "string" || fieldType === "expression" || fieldType === "text" || fieldType === "xml";
}

function isDateTimeField(fieldType: string | undefined): boolean {
  return (
    fieldType === "date" ||
    fieldType === "datetime" ||
    fieldType === "time" ||
    fieldType === "time-range" ||
    fieldType === "day-in-year" ||
    fieldType === "cron"
  );
}

function isReferenceField(fieldType: string | undefined): boolean {
  return (
    fieldType === "git-ref" ||
    fieldType === "repository-file" ||
    fieldType === "timezone" ||
    fieldType === "any-predicate-list"
  );
}
