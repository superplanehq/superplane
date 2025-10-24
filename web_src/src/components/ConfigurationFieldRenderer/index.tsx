import React from 'react'
import { Label } from '../ui/label'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Tooltip, TooltipTrigger, TooltipContent } from '../ui/tooltip'
import { FieldRendererProps } from './types'
import { StringFieldRenderer } from './StringFieldRenderer'
import { NumberFieldRenderer } from './NumberFieldRenderer'
import { BooleanFieldRenderer } from './BooleanFieldRenderer'
import { SelectFieldRenderer } from './SelectFieldRenderer'
import { MultiSelectFieldRenderer } from './MultiSelectFieldRenderer'
import { DateFieldRenderer } from './DateFieldRenderer'
import { UrlFieldRenderer } from './UrlFieldRenderer'
import { ListFieldRenderer } from './ListFieldRenderer'
import { ObjectFieldRenderer } from './ObjectFieldRenderer'
import { IntegrationFieldRenderer } from './IntegrationFieldRenderer'
import { IntegrationResourceFieldRenderer } from './IntegrationResourceFieldRenderer'
import { TimeFieldRenderer } from './TimeFieldRenderer'
import { isFieldVisible } from '../../utils/components'

interface ConfigurationFieldRendererProps extends FieldRendererProps {
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
    return isFieldVisible(field, allValues)
  }, [field, allValues])

  if (!isVisible) {
    return null
  }
  const renderField = () => {
    const commonProps = { field, value, onChange, allValues }

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

      case 'url':
        return <UrlFieldRenderer {...commonProps} />

      case 'time':
        return <TimeFieldRenderer {...commonProps} />

      case 'integration':
        return <IntegrationFieldRenderer field={field} value={value} onChange={onChange} domainId={domainId} domainType={domainType} />

      case 'integration-resource':
        return <IntegrationResourceFieldRenderer field={field} value={value} onChange={onChange} allValues={allValues} domainId={domainId} domainType={domainType} />

      case 'list':
        return <ListFieldRenderer {...commonProps} />

      case 'object':
        return <ObjectFieldRenderer {...commonProps} />

      default:
        // Fallback to text input
        return <StringFieldRenderer {...commonProps} />
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
