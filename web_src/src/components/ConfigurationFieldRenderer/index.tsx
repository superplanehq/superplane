import React from 'react'
import Editor from '@monaco-editor/react'
import { ComponentsConfigurationField } from '../../api-client'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { Button } from '../ui/button'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Tooltip, TooltipTrigger, TooltipContent } from '../ui/tooltip'
import { useIntegrations, useIntegrationResources } from '../../pages/canvas/hooks/useIntegrations'

interface ConfigurationFieldRendererProps {
  field: ComponentsConfigurationField
  value: any
  onChange: (value: any) => void
  allValues?: Record<string, any>
  domainId?: string
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION"
}

export const ConfigurationFieldRenderer = ({
  field,
  value,
  onChange,
  allValues = {},
  domainId,
  domainType
}: ConfigurationFieldRendererProps) => {
  // Check visibility conditions
  const isVisible = React.useMemo(() => {
    if (!field.visibilityConditions || field.visibilityConditions.length === 0) {
      return true
    }

    // All conditions must be satisfied (AND logic)
    return field.visibilityConditions.every((condition) => {
      if (!condition.field || !condition.values) {
        return true
      }

      const fieldValue = allValues[condition.field]

      // Convert field value to string for comparison
      const fieldValueStr = fieldValue !== undefined && fieldValue !== null
        ? String(fieldValue)
        : ''

      // Check if the field value matches any of the expected values
      // Support wildcard "*" to match any non-empty value
      return condition.values.some((expectedValue) => {
        if (expectedValue === '*') {
          // Wildcard matches any non-empty value
          return fieldValueStr !== ''
        }
        return fieldValueStr === expectedValue
      })
    })
  }, [field.visibilityConditions, allValues])

  if (!isVisible) {
    return null
  }
  const renderField = () => {
    switch (field.type) {
      case 'string':
        return (
          <Input
            type="text"
            value={value ?? field.defaultValue ?? ''}
            onChange={(e) => onChange(e.target.value || undefined)}
            placeholder={`Enter ${field.name}`}
          />
        )

      case 'number':
        const numberOptions = field.typeOptions?.number
        return (
          <Input
            type="number"
            value={value ?? field.defaultValue ?? ''}
            onChange={(e) => {
              const val = e.target.value === '' ? undefined : Number(e.target.value)
              onChange(val)
            }}
            placeholder={`Enter ${field.name}`}
            min={numberOptions?.min}
            max={numberOptions?.max}
          />
        )

      case 'boolean':
        return (
          <input
            type="checkbox"
            checked={value ?? (field.defaultValue === 'true') ?? false}
            onChange={(e) => onChange(e.target.checked)}
            className="h-4 w-4 rounded border-gray-300 dark:border-zinc-700"
          />
        )

      case 'select':
        const selectOptions = field.typeOptions?.select?.options ?? []
        return (
          <Select
            value={value ?? field.defaultValue ?? ''}
            onValueChange={(val) => onChange(val || undefined)}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder={`Select ${field.label || field.name}`} />
            </SelectTrigger>
            <SelectContent>
              {selectOptions.map((opt) => (
                <SelectItem key={opt.value} value={opt.value ?? ''}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )

      case 'multi_select':
        const multiSelectOptions = field.typeOptions?.multiSelect?.options ?? []
        return (
          <select
            multiple
            value={value ?? (field.defaultValue ? JSON.parse(field.defaultValue) : [])}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              const selected = Array.from(e.target.selectedOptions, opt => opt.value)
              onChange(selected.length > 0 ? selected : undefined)
            }}
            className="w-full px-3 py-2 border border-gray-300 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100"
            size={Math.min(multiSelectOptions.length ?? 5, 5)}
          >
            {multiSelectOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        )

      case 'date':
        return (
          <Input
            type="date"
            value={value ?? field.defaultValue ?? ''}
            onChange={(e) => onChange(e.target.value || undefined)}
          />
        )

      case 'url':
        return (
          <Input
            type="url"
            value={value ?? field.defaultValue ?? ''}
            onChange={(e) => onChange(e.target.value || undefined)}
            placeholder="https://example.com"
          />
        )

      case 'integration':
        return <IntegrationFieldRenderer field={field} value={value} onChange={onChange} domainId={domainId} domainType={domainType} />

      case 'integration-resource':
        return <IntegrationResourceFieldRenderer field={field} value={value} onChange={onChange} allValues={allValues} domainId={domainId} domainType={domainType} />

      case 'list':
        return <ListFieldRenderer field={field} value={value} onChange={onChange} />

      case 'object':
        return <ObjectFieldRenderer field={field} value={value} onChange={onChange} />

      default:
        // Fallback to text input
        return (
          <Input
            type="text"
            value={value ?? field.defaultValue ?? ''}
            onChange={(e) => onChange(e.target.value || undefined)}
            placeholder={`Enter ${field.name}`}
          />
        )
    }
  }

  return (
    <div className="space-y-2">
      <Label className="block text-left">
        {field.label || field.name}
      </Label>
      <div className="flex items-center gap-2">
        <div className="flex-1">
          {renderField()}
        </div>
        {field.description && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button type="button" className="text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-300">
                <MaterialSymbol name="info" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="top">
              <p className="max-w-xs">{field.description}</p>
            </TooltipContent>
          </Tooltip>
        )}
      </div>
      {field.typeOptions?.number?.min !== undefined && field.typeOptions?.number?.max !== undefined && (
        <p className="text-xs text-gray-500 dark:text-zinc-400 text-left">
          Range: {field.typeOptions.number.min} - {field.typeOptions.number.max}
        </p>
      )}
    </div>
  )
}

// Integration field renderer with select dropdown
const IntegrationFieldRenderer = ({
  field,
  value,
  onChange,
  domainId,
  domainType
}: {
  field: ComponentsConfigurationField
  value: string
  onChange: (value: string | undefined) => void
  domainId?: string
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION"
}) => {
  const integrationType = field.typeOptions?.integration?.type

  // Fetch integrations if we have the required context
  const { data: integrations, isLoading } = useIntegrations(
    domainId ?? '',
    domainType ?? 'DOMAIN_TYPE_CANVAS'
  )

  // Filter integrations by type
  const filteredIntegrations = React.useMemo(() => {
    if (!integrations || !integrationType) return []
    return integrations.filter((integration) => integration.spec?.type === integrationType)
  }, [integrations, integrationType])

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Integration field requires domainId and domainType props
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="text-sm text-gray-500 dark:text-zinc-400">
        Loading {integrationType} integrations...
      </div>
    )
  }

  if (filteredIntegrations.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No integrations available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          No {integrationType} integrations found. Please create one in the settings.
        </p>
      </div>
    )
  }

  return (
    <Select
      value={value ?? ''}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder={`Select ${integrationType} integration`} />
      </SelectTrigger>
      <SelectContent>
        {filteredIntegrations.map((integration) => (
          <SelectItem key={integration.metadata?.id} value={integration.metadata?.id ?? ''}>
            {integration.metadata?.name} ({integration.spec?.type})
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

// Integration resource field renderer with select dropdown for resources
const IntegrationResourceFieldRenderer = ({
  field,
  value,
  onChange,
  allValues,
  domainId,
  domainType
}: {
  field: ComponentsConfigurationField
  value: string
  onChange: (value: string | undefined) => void
  allValues?: Record<string, any>
  domainId?: string
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION"
}) => {
  const resourceType = field.typeOptions?.resource?.type

  // Find the integration field by looking at the visibility conditions
  // The integration field should be referenced in the visibility condition
  const integrationFieldName = React.useMemo(() => {
    if (!field.visibilityConditions || field.visibilityConditions.length === 0) {
      return undefined
    }
    // Find the first visibility condition that references a field (should be the integration field)
    const condition = field.visibilityConditions.find(c => c.field)
    return condition?.field
  }, [field.visibilityConditions])

  // Get the selected integration ID from allValues
  const selectedIntegrationId = integrationFieldName ? allValues?.[integrationFieldName] : undefined

  // Fetch integrations to get the selected integration details
  const { data: integrations } = useIntegrations(
    domainId ?? '',
    domainType ?? 'DOMAIN_TYPE_CANVAS'
  )

  // Find the selected integration
  const selectedIntegration = React.useMemo(() => {
    if (!integrations || !selectedIntegrationId) return null
    return integrations.find((integration) => integration.metadata?.id === selectedIntegrationId)
  }, [integrations, selectedIntegrationId])

  // Fetch resources using the hook
  const { data: resources, isLoading: isLoadingResources, error: resourcesError } = useIntegrationResources(
    domainId ?? '',
    domainType ?? 'DOMAIN_TYPE_CANVAS',
    selectedIntegrationId ?? '',
    resourceType ?? ''
  )

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Integration resource field requires domainId and domainType props
      </div>
    )
  }

  if (!selectedIntegrationId) {
    return (
      <Select disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={`Select ${resourceType} integration first`} />
        </SelectTrigger>
      </Select>
    )
  }

  if (isLoadingResources) {
    return (
      <div className="text-sm text-gray-500 dark:text-zinc-400">
        Loading {resourceType} resources...
      </div>
    )
  }

  if (resourcesError) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load resources
      </div>
    )
  }

  if (!resources || resources.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No resources available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          No {resourceType} resources found in {selectedIntegration?.metadata?.name}
        </p>
      </div>
    )
  }

  return (
    <Select
      value={value ?? ''}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder={`Select ${resourceType}`} />
      </SelectTrigger>
      <SelectContent>
        {resources.map((resource) => (
          <SelectItem key={resource.name} value={resource.name}>
            {resource.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

// List field renderer with add/remove functionality
const ListFieldRenderer = ({
  field,
  value,
  onChange
}: {
  field: ComponentsConfigurationField
  value: any
  onChange: (value: any) => void
}) => {
  const items = Array.isArray(value) ? value : []
  const listOptions = field.typeOptions?.list
  const itemDefinition = listOptions?.itemDefinition

  const addItem = () => {
    const newItem = itemDefinition?.type === 'object'
      ? {}
      : itemDefinition?.type === 'number'
      ? 0
      : ''
    onChange([...items, newItem])
  }

  const removeItem = (index: number) => {
    const newItems = items.filter((_, i) => i !== index)
    onChange(newItems.length > 0 ? newItems : undefined)
  }

  const updateItem = (index: number, newValue: any) => {
    const newItems = [...items]
    newItems[index] = newValue
    onChange(newItems)
  }

  return (
    <div className="space-y-3">
      {items.map((item, index) => (
        <div key={index} className="flex gap-2 items-center">
          <div className="flex-1">
            {itemDefinition?.type === 'object' && itemDefinition.schema ? (
              <div className="border border-gray-300 dark:border-zinc-700 rounded-md p-4 space-y-4">
                {itemDefinition.schema.map((schemaField) => (
                  <ConfigurationFieldRenderer
                    key={schemaField.name}
                    field={schemaField}
                    value={item[schemaField.name!]}
                    onChange={(val) => {
                      const newItem = { ...item, [schemaField.name!]: val }
                      updateItem(index, newItem)
                    }}
                    allValues={item}
                  />
                ))}
              </div>
            ) : (
              <Input
                type={itemDefinition?.type === 'number' ? 'number' : 'text'}
                value={item ?? ''}
                onChange={(e) => {
                  const val = itemDefinition?.type === 'number'
                    ? (e.target.value === '' ? undefined : Number(e.target.value))
                    : e.target.value
                  updateItem(index, val)
                }}
              />
            )}
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => removeItem(index)}
            className="mt-1"
          >
            <MaterialSymbol name="delete" className="text-red-500" />
          </Button>
        </div>
      ))}
      <Button
        variant="outline"
        onClick={addItem}
        className="w-full mt-3"
      >
        <MaterialSymbol name="add" />
        Add Item
      </Button>
    </div>
  )
}

// Object field renderer with nested fields
const ObjectFieldRenderer = ({
  field,
  value,
  onChange
}: {
  field: ComponentsConfigurationField
  value: Record<string, any>
  onChange: (value: Record<string, any>) => void
}) => {
  const objValue = value ?? {}
  const [isDarkMode, setIsDarkMode] = React.useState(false)
  const [jsonError, setJsonError] = React.useState<string | null>(null)
  const objectOptions = field.typeOptions?.object
  const schema = objectOptions?.schema

  // Detect dark mode
  React.useEffect(() => {
    const checkDarkMode = () => {
      setIsDarkMode(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)
    }

    checkDarkMode()

    const observer = new MutationObserver(checkDarkMode)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    })

    return () => observer.disconnect()
  }, [])

  if (!schema || schema.length === 0) {
    // Fallback to Monaco Editor if no schema defined
    const handleEditorChange = (value: string | undefined) => {
      const newValue = value || '{}'
      try {
        const parsed = JSON.parse(newValue)
        onChange(parsed)
        setJsonError(null)
      } catch (error) {
        setJsonError('Invalid JSON format')
      }
    }

    return (
      <div className="flex flex-col gap-2">
        <div className="border border-gray-300 dark:border-zinc-700 rounded-md overflow-hidden" style={{ height: '200px' }}>
          <Editor
            height="100%"
            defaultLanguage="json"
            value={JSON.stringify(objValue, null, 2)}
            onChange={handleEditorChange}
            theme={isDarkMode ? 'vs-dark' : 'vs'}
            options={{
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: 'on',
              wordWrap: 'on',
              folding: true,
              bracketPairColorization: {
                enabled: true
              },
              autoIndent: 'advanced',
              formatOnPaste: true,
              formatOnType: true,
              tabSize: 2,
              insertSpaces: true,
              scrollBeyondLastLine: false,
              renderWhitespace: 'boundary',
              smoothScrolling: true,
              cursorBlinking: 'smooth',
              contextmenu: true,
              selectOnLineNumbers: true
            }}
          />
        </div>
        {jsonError && (
          <p className="text-red-600 dark:text-red-400 text-xs">
            {jsonError}
          </p>
        )}
      </div>
    )
  }

  return (
    <div className="border border-gray-300 dark:border-zinc-700 rounded-md p-4 space-y-4">
      {schema.map((schemaField) => (
        <ConfigurationFieldRenderer
          key={schemaField.name}
          field={schemaField}
          value={objValue[schemaField.name!]}
          onChange={(val) => {
            const newValue = { ...objValue, [schemaField.name!]: val }
            onChange(newValue)
          }}
          allValues={objValue}
        />
      ))}
    </div>
  )
}
