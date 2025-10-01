import type { SuperplaneTypeManifest } from '../../api-client';
import { DynamicFormField } from './DynamicFormField';
import type { DynamicFormContext } from './types';

interface DynamicFormProps {
  manifest: SuperplaneTypeManifest;
  value: Record<string, any>;
  onChange: (value: Record<string, any>) => void;
  disabled?: boolean;
  errors?: Record<string, string>;
  context?: DynamicFormContext;
}

export function DynamicForm({
  manifest,
  value,
  onChange,
  disabled = false,
  errors = {},
  context,
}: DynamicFormProps) {
  const handleFieldChange = (fieldName: string, fieldValue: any) => {
    onChange({
      ...value,
      [fieldName]: fieldValue,
    });
  };

  if (!manifest.fields || manifest.fields.length === 0) {
    return (
      <div className="text-sm text-zinc-500">
        No configuration required for {manifest.displayName || manifest.type}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {manifest.fields.map((field) => (
        <DynamicFormField
          key={field.name}
          field={field}
          value={value[field.name || '']}
          onChange={(newValue) => handleFieldChange(field.name || '', newValue)}
          disabled={disabled}
          error={errors[field.name || '']}
          context={context}
        />
      ))}
    </div>
  );
}
