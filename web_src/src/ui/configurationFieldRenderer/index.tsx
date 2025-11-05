import React from 'react'
import { Info } from 'lucide-react'
import { Label } from '../label'
import { Tooltip, TooltipTrigger, TooltipContent } from '../tooltip'
import { FieldRendererProps } from './types'
import { StringFieldRenderer } from './StringFieldRenderer'
import { NumberFieldRenderer } from './NumberFieldRenderer'
import { BooleanFieldRenderer } from './BooleanFieldRenderer'
import { SelectFieldRenderer } from './SelectFieldRenderer'
import { MultiSelectFieldRenderer } from './MultiSelectFieldRenderer'
import { DateFieldRenderer } from './DateFieldRenderer'
import { DateTimeFieldRenderer } from './DateTimeFieldRenderer'
import { UrlFieldRenderer } from './UrlFieldRenderer'
import { ListFieldRenderer } from './ListFieldRenderer'
import { ObjectFieldRenderer } from './ObjectFieldRenderer'
import { IntegrationFieldRenderer } from './IntegrationFieldRenderer'
import { IntegrationResourceFieldRenderer } from './IntegrationResourceFieldRenderer'
import { TimeFieldRenderer } from './TimeFieldRenderer'
import { UserFieldRenderer } from './UserFieldRenderer'
import { RoleFieldRenderer } from './RoleFieldRenderer'
import { GroupFieldRenderer } from './GroupFieldRenderer'
import { isFieldVisible } from '../../utils/components'

interface ConfigurationFieldRendererProps extends FieldRendererProps {
  domainId?: string
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION"
  hasError?: boolean
  validationErrors?: Set<string>
  fieldPath?: string
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
  fieldPath
}: ConfigurationFieldRendererProps) => {
  // Check visibility conditions
  const isVisible = React.useMemo(() => {
    return isFieldVisible(field, allValues)
  }, [field, allValues])

  if (!isVisible) {
    return null
  }
  const renderField = () => {
    const commonProps = { field, value, onChange, allValues, hasError }

    switch (field.type) {
      case 'string':
        return <StringFieldRenderer {...commonProps} />

      case 'number':
        return <NumberFieldRenderer {...commonProps} />

      case 'boolean':
        return <BooleanFieldRenderer {...commonProps} />

      case 'select':
        return <SelectFieldRenderer {...commonProps} />

      case 'multi-select':
        return <MultiSelectFieldRenderer {...commonProps} />

      case 'date':
        return <DateFieldRenderer {...commonProps} />

      case 'datetime':
        return <DateTimeFieldRenderer {...commonProps} />

      case 'url':
        return <UrlFieldRenderer {...commonProps} />

      case 'time':
        return <TimeFieldRenderer {...commonProps} />

      case 'integration':
        return <IntegrationFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} domainType={domainType} />

      case 'integration-resource':
        return <IntegrationResourceFieldRenderer field={field} value={value as string} onChange={onChange} allValues={allValues} domainId={domainId} domainType={domainType} />

      case 'user':
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">User field requires domainId prop</div>
        }
        return <UserFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} />

      case 'role':
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">Role field requires domainId prop</div>
        }
        return <RoleFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} />

      case 'group':
        if (!domainId) {
          return <div className="text-sm text-red-500 dark:text-red-400">Group field requires domainId prop</div>
        }
        return <GroupFieldRenderer field={field} value={value as string} onChange={onChange} domainId={domainId} />

      case 'list':
        return <ListFieldRenderer {...commonProps} domainId={domainId} domainType={domainType} validationErrors={validationErrors} fieldPath={fieldPath || field.name} />

      case 'object':
        return <ObjectFieldRenderer {...commonProps} domainId={domainId} domainType={domainType} />

      default:
        // Fallback to text input
        return <StringFieldRenderer {...commonProps} />
    }
  }

  return (
    <div className="space-y-2">
      <Label className={`block text-left ${hasError ? 'text-red-600 dark:text-red-400' : ''}`}>
        {field.label || field.name}
        {field.required && <span className="text-red-500 ml-1">*</span>}
        {hasError && field.required && (
          <span className="text-red-500 text-xs ml-2">- required field</span>
        )}
      </Label>
      <div className="flex items-center gap-2">
        <div className="flex-1">
          {renderField()}
        </div>
        {field.description && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button type="button" className="text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-300">
                <Info className="h-4 w-4" />
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
