import { Eye, EyeOff } from "lucide-react";
import { useState, type ReactNode } from "react";
import type { ConfigurationField, ConfigurationSelectOption } from "@/api-client";
import { cn } from "@/lib/utils";
import type { ReadonlyExpressionPreview } from "./expressionPreview";

export function ReadonlyConfigurationField({
  field,
  label,
  description,
  value,
  isTogglable,
  isEnabled,
  expressionPreview,
}: {
  field: ConfigurationField;
  label?: string;
  description?: string;
  value: unknown;
  isTogglable: boolean;
  isEnabled: boolean;
  expressionPreview?: ReadonlyExpressionPreview | null;
}) {
  const [isPreviewVisible, setIsPreviewVisible] = useState(true);
  const hasExpressionError = expressionPreview?.status === "error";
  const shouldShowPreviewToggle = expressionPreview?.status === "resolved";

  if (isTogglable && !isEnabled) {
    return (
      <div className="space-y-2 opacity-70">
        <ReadonlyFieldLabel label={label} />
        <ReadonlyPrimitiveValue value="Disabled" />
      </div>
    );
  }

  if (field.type === "boolean" && !expressionPreview) {
    return <ReadonlyBooleanField label={label} value={value === true} description={description} />;
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        <ReadonlyFieldLabel label={label} />
        {shouldShowPreviewToggle ? (
          <ReadonlyPreviewToggle isPreviewVisible={isPreviewVisible} onToggle={setIsPreviewVisible} />
        ) : null}
      </div>
      <ReadonlyValue
        field={field}
        value={value}
        hasExpressionError={hasExpressionError}
        expressionTemplate={expressionPreview?.templateValue}
        resolvedValue={expressionPreview?.status === "resolved" ? expressionPreview.value : undefined}
        isPreviewVisible={isPreviewVisible}
      />
      {expressionPreview?.status === "error" ? (
        <ReadonlyExpressionErrorMessage fieldName={field.name} message={expressionPreview.value} />
      ) : null}
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

function ReadonlyPreviewToggle({
  isPreviewVisible,
  onToggle,
}: {
  isPreviewVisible: boolean;
  onToggle: (isVisible: boolean) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onToggle(!isPreviewVisible)}
      className={cn(
        "ml-auto flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px] font-medium transition-colors",
        isPreviewVisible
          ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300"
          : "text-emerald-600 hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/30",
      )}
    >
      {isPreviewVisible ? <Eye className="h-3 w-3" /> : <EyeOff className="h-3 w-3" />}
      <span>{isPreviewVisible ? "Show expression" : "Show applied"}</span>
    </button>
  );
}

function ReadonlyValue({
  field,
  value,
  hasExpressionError = false,
  expressionTemplate,
  resolvedValue,
  isPreviewVisible,
}: {
  field: ConfigurationField;
  value: unknown;
  hasExpressionError?: boolean;
  expressionTemplate?: string;
  resolvedValue?: string;
  isPreviewVisible: boolean;
}) {
  const rawValue = formatReadonlyValue(field, value);
  const isResolvedPreview = isPreviewVisible && resolvedValue !== undefined;
  const displayValue = isResolvedPreview ? resolvedValue : expressionTemplate || rawValue;
  const className = hasExpressionError
    ? "border-red-300 dark:border-red-800"
    : isResolvedPreview
      ? "border-emerald-300 bg-emerald-50/40 dark:border-emerald-800 dark:bg-emerald-950/20"
      : undefined;

  if (isStructuredReadonlyValue(value)) {
    return (
      <pre
        className={cn(
          "max-h-56 overflow-auto rounded-md border border-slate-200 bg-white p-2 text-xs text-slate-900 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100",
          className,
        )}
      >
        {displayValue}
      </pre>
    );
  }

  return (
    <ReadonlyPrimitiveValue value={displayValue} className={className}>
      {isResolvedPreview
        ? renderAppliedExpressionValue(field, displayValue, expressionTemplate)
        : renderReadonlyHighlightedValue(field, displayValue, hasExpressionError)}
    </ReadonlyPrimitiveValue>
  );
}

function ReadonlyPrimitiveValue({
  value,
  className,
  children,
}: {
  value: string;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "min-h-9 w-full whitespace-pre-wrap break-words rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100",
        className,
      )}
    >
      {children ?? value}
    </div>
  );
}

function renderReadonlyHighlightedValue(
  field: ConfigurationField,
  value: string,
  hasExpressionError: boolean,
): ReactNode {
  if (!value) return value;

  if (field.type === "expression") {
    return <ReadonlyExpressionSegment expression={value} hasError={hasExpressionError} />;
  }

  if (!hasWrappedExpression(value)) return value;

  const parts: ReactNode[] = [];
  const expressionPattern = /(\{\{)([\s\S]*?)(\}\})/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  let key = 0;

  while ((match = expressionPattern.exec(value)) !== null) {
    if (match.index > lastIndex) {
      parts.push(<span key={key++}>{value.slice(lastIndex, match.index)}</span>);
    }

    parts.push(
      <ReadonlyExpressionSegment
        key={key++}
        expression={match[2]}
        prefix={match[1]}
        suffix={match[3]}
        hasError={hasExpressionError}
      />,
    );
    lastIndex = expressionPattern.lastIndex;
  }

  if (lastIndex < value.length) {
    parts.push(<span key={key++}>{value.slice(lastIndex)}</span>);
  }

  return parts;
}

function ReadonlyExpressionSegment({
  expression,
  prefix,
  suffix,
  hasError,
}: {
  expression: string;
  prefix?: string;
  suffix?: string;
  hasError: boolean;
}) {
  if (hasError) {
    return (
      <span className="font-medium text-red-600 underline decoration-red-300 decoration-dotted underline-offset-2 dark:text-red-400 dark:decoration-red-700">
        {prefix}
        {expression}
        {suffix}
      </span>
    );
  }

  return (
    <span className="rounded-sm bg-slate-100 dark:bg-gray-800">
      {prefix ? <span className="text-gray-400 dark:text-gray-500">{prefix}</span> : null}
      <span className="font-medium text-violet-700 dark:text-violet-300">{expression}</span>
      {suffix ? <span className="text-gray-400 dark:text-gray-500">{suffix}</span> : null}
    </span>
  );
}

function hasWrappedExpression(value: string): boolean {
  return /\{\{[\s\S]*?\}\}/.test(value);
}

function renderAppliedExpressionValue(
  field: ConfigurationField,
  value: string,
  expressionTemplate?: string,
): ReactNode {
  if (field.type === "expression") {
    return <ReadonlyResolvedSegment value={value} />;
  }

  if (!expressionTemplate) {
    return <ReadonlyResolvedSegment value={value} />;
  }

  const highlightedValue = renderResolvedWrappedExpressionValue(value, expressionTemplate);
  return highlightedValue ?? value;
}

function renderResolvedWrappedExpressionValue(value: string, expressionTemplate: string): ReactNode[] | null {
  const templateParts = parseWrappedExpressionTemplate(expressionTemplate);
  if (!templateParts.some((part) => part.type === "expression")) return null;

  const renderedParts: ReactNode[] = [];
  let valueIndex = 0;
  let key = 0;

  for (let index = 0; index < templateParts.length; index++) {
    const part = templateParts[index];

    if (part.type === "literal") {
      if (!value.startsWith(part.value, valueIndex)) return null;
      renderedParts.push(<span key={key++}>{part.value}</span>);
      valueIndex += part.value.length;
      continue;
    }

    const nextLiteral = findNextLiteral(templateParts, index);
    const expressionEnd = nextLiteral ? value.indexOf(nextLiteral, valueIndex) : value.length;
    if (expressionEnd < valueIndex) return null;

    const resolvedValue = value.slice(valueIndex, expressionEnd);
    renderedParts.push(<ReadonlyResolvedSegment key={key++} value={resolvedValue} />);
    valueIndex = expressionEnd;
  }

  if (valueIndex !== value.length) return null;
  return renderedParts;
}

type WrappedExpressionTemplatePart =
  | {
      type: "literal";
      value: string;
    }
  | {
      type: "expression";
      value: string;
    };

function parseWrappedExpressionTemplate(template: string): WrappedExpressionTemplatePart[] {
  const parts: WrappedExpressionTemplatePart[] = [];
  const expressionPattern = /\{\{([\s\S]*?)\}\}/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = expressionPattern.exec(template)) !== null) {
    parts.push({ type: "literal", value: template.slice(lastIndex, match.index) });
    parts.push({ type: "expression", value: match[1] });
    lastIndex = expressionPattern.lastIndex;
  }

  parts.push({ type: "literal", value: template.slice(lastIndex) });
  return parts;
}

function findNextLiteral(parts: WrappedExpressionTemplatePart[], startIndex: number): string | null {
  for (let index = startIndex + 1; index < parts.length; index++) {
    const part = parts[index];
    if (part.type === "literal" && part.value) return part.value;
  }

  return null;
}

function ReadonlyResolvedSegment({ value }: { value: string }) {
  return (
    <span className="rounded-sm bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300">
      <span className="text-emerald-500 dark:text-emerald-500">{"{{ "}</span>
      {value}
      <span className="text-emerald-500 dark:text-emerald-500">{" }}"}</span>
    </span>
  );
}

function ReadonlyExpressionErrorMessage({ fieldName, message }: { fieldName?: string; message: string }) {
  return (
    <p
      data-testid={fieldName ? `runtime-config-expression-error-${fieldName}` : undefined}
      className="text-xs leading-normal text-red-600 dark:text-red-400"
    >
      {message}
    </p>
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
