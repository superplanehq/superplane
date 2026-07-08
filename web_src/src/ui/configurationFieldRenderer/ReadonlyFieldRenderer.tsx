import type { ConfigurationField, ConfigurationSelectOption } from "@/api-client";

export function ReadonlyConfigurationField({
  field,
  label,
  description,
  value,
  isTogglable,
  isEnabled,
}: {
  field: ConfigurationField;
  label?: string;
  description?: string;
  value: unknown;
  isTogglable: boolean;
  isEnabled: boolean;
}) {
  if (isTogglable && !isEnabled) {
    return (
      <div className="space-y-2 opacity-70">
        <ReadonlyFieldLabel label={label} />
        <ReadonlyPrimitiveValue value="Disabled" />
      </div>
    );
  }

  if (field.type === "boolean") {
    return <ReadonlyBooleanField label={label} value={value === true} description={description} />;
  }

  return (
    <div className="space-y-2">
      <ReadonlyFieldLabel label={label} />
      <ReadonlyValue field={field} value={value} />
      {description ? <p className="text-xs leading-normal text-gray-500 dark:text-gray-400">{description}</p> : null}
    </div>
  );
}

function ReadonlyBooleanField({ label, value, description }: { label?: string; value: boolean; description?: string }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        <span
          className={
            value
              ? "relative inline-flex h-5 w-9 rounded-full bg-blue-500"
              : "relative inline-flex h-5 w-9 rounded-full bg-slate-200 dark:bg-gray-700"
          }
          aria-hidden="true"
        >
          <span
            className={
              value
                ? "absolute right-0.5 top-0.5 h-4 w-4 rounded-full bg-white"
                : "absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white"
            }
          />
        </span>
        <ReadonlyFieldLabel label={label} />
      </div>
      {description ? <p className="text-xs leading-normal text-gray-500 dark:text-gray-400">{description}</p> : null}
    </div>
  );
}

function ReadonlyFieldLabel({ label }: { label?: string }) {
  return <div className="text-sm font-medium text-slate-800 dark:text-gray-100">{label}</div>;
}

function ReadonlyValue({ field, value }: { field: ConfigurationField; value: unknown }) {
  const displayValue = formatReadonlyValue(field, value);

  if (isStructuredReadonlyValue(value)) {
    return (
      <pre className="max-h-56 overflow-auto rounded-md border border-slate-200 bg-white p-2 text-xs text-slate-900 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100">
        {displayValue}
      </pre>
    );
  }

  return <ReadonlyPrimitiveValue value={displayValue} />;
}

function ReadonlyPrimitiveValue({ value }: { value: string }) {
  return (
    <div className="min-h-9 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100">
      {value}
    </div>
  );
}

function formatReadonlyValue(field: ConfigurationField, value: unknown): string {
  if (field.type === "select" && typeof value === "string") {
    return findOptionLabel(field.typeOptions?.select?.options, value);
  }

  if (field.type === "multi-select" || field.type === "days-of-week") {
    const values = Array.isArray(value) ? value : [];
    return values.map((item) => findOptionLabel(field.typeOptions?.multiSelect?.options, String(item))).join(", ");
  }

  if (value === null || value === undefined || value === "") {
    return "";
  }

  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return JSON.stringify(value, null, 2);
}

function findOptionLabel(options: ConfigurationSelectOption[] | undefined, value: string): string {
  return options?.find((option) => option.value === value)?.label || value;
}

function isStructuredReadonlyValue(value: unknown): boolean {
  return value !== null && typeof value === "object";
}
