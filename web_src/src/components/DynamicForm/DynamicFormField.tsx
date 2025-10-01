import { useState, useEffect } from 'react';
import { Input } from '../Input/input';
import { AutoCompleteSelect } from '../AutoCompleteSelect/AutoCompleteSelect';
import { MapField } from './MapField';
import { ArrayField } from './ArrayField';
import { integrationsListResources } from '@/api-client';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';
import type { DynamicFormFieldProps } from './types';

export function DynamicFormField({
  field,
  value,
  onChange,
  parentPath = '',
  disabled = false,
  error,
  context,
}: DynamicFormFieldProps) {
  const fieldPath = parentPath ? `${parentPath}.${field.name}` : field.name || '';
  const [dynamicOptions, setDynamicOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [loadingOptions, setLoadingOptions] = useState(false);

  // Load dynamic options for resource fields
  useEffect(() => {
    if (field.type === 'resource' && field.resourceType && context?.integrationName && context?.canvasId) {
      setLoadingOptions(true);

      integrationsListResources(withOrganizationHeader({
        path: {
          idOrName: context.integrationName,
        },
        query: {
          domainType: 'DOMAIN_TYPE_CANVAS',
          domainId: context.canvasId,
          type: field.resourceType,
        },
      }))
        .then((response) => {
          const resources = response.data?.resources || [];
          const options = resources.map((item) => ({
            value: item.name || item.id || '',
            label: item.name || item.id || '',
          }));
          setDynamicOptions(options);
        })
        .catch((err) => {
          console.error('Error loading dynamic options:', err);
          setDynamicOptions([]);
        })
        .finally(() => {
          setLoadingOptions(false);
        });
    }
  }, [field.type, field.resourceType, context?.integrationName, context?.canvasId]);

  // Handle hidden fields
  if (field.hidden) {
    return null;
  }

  // Render field based on type
  const renderField = () => {
    switch (field.type) {
      case 'string':
      case 'number':
        return (
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <Input
              type={field.type === 'number' ? 'number' : 'text'}
              value={value || ''}
              onChange={(e) => {
                const newValue = field.type === 'number'
                  ? e.target.value ? Number(e.target.value) : undefined
                  : e.target.value;
                onChange(newValue);
              }}
              placeholder={field.placeholder}
              disabled={disabled}
              data-invalid={!!error}
            />
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      case 'boolean':
        return (
          <div className="space-y-1">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={value || false}
                onChange={(e) => onChange(e.target.checked)}
                disabled={disabled}
                className="rounded border-zinc-300"
              />
              <span className="text-sm font-medium">
                {field.displayName || field.name}
                {field.required && <span className="text-red-500 ml-1">*</span>}
              </span>
            </label>
            {field.description && (
              <p className="text-sm text-zinc-500 ml-6">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500 ml-6">{error}</p>}
          </div>
        );

      case 'select':
        const selectOptions = field.options?.map((opt) => ({
          value: opt.value || '',
          label: opt.label || '',
        })) || [];

        return (
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <AutoCompleteSelect
              value={value || ''}
              onChange={(newValue) => onChange(newValue)}
              options={selectOptions}
              placeholder={field.placeholder}
              disabled={disabled}
            />
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      case 'resource':
        return (
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <AutoCompleteSelect
              value={value || ''}
              onChange={(newValue) => onChange(newValue)}
              options={dynamicOptions}
              placeholder={loadingOptions ? 'Loading...' : field.placeholder}
              disabled={disabled || loadingOptions}
            />
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      case 'textarea':
        return (
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <textarea
              value={value || ''}
              onChange={(e) => onChange(e.target.value)}
              placeholder={field.placeholder}
              disabled={disabled}
              rows={4}
              className="w-full rounded-lg border border-zinc-300 px-3 py-2 text-sm"
            />
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      case 'map':
        return (
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <MapField
              value={value || {}}
              onChange={onChange}
              placeholder={field.placeholder}
              disabled={disabled}
            />
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      case 'array':
        return (
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <ArrayField
              field={field}
              value={value || []}
              onChange={onChange}
              disabled={disabled}
              parentPath={fieldPath}
              context={context}
            />
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      case 'object':
        return (
          <div className="space-y-3">
            <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {field.displayName || field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            {field.description && (
              <p className="text-sm text-zinc-500">{field.description}</p>
            )}
            <div className="pl-4 border-l-2 border-zinc-200 space-y-3">
              {field.fields?.map((nestedField) => (
                <DynamicFormField
                  key={nestedField.name}
                  field={nestedField}
                  value={value?.[nestedField.name || '']}
                  onChange={(newValue) => {
                    onChange({
                      ...(value || {}),
                      [nestedField.name || '']: newValue,
                    });
                  }}
                  parentPath={fieldPath}
                  disabled={disabled}
                  context={context}
                />
              ))}
            </div>
            {error && <p className="text-sm text-red-500">{error}</p>}
          </div>
        );

      default:
        return (
          <div className="text-sm text-zinc-500">
            Unsupported field type: {field.type}
          </div>
        );
    }
  };

  return <div className="space-y-2">{renderField()}</div>;
}
