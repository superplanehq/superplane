import React, { useEffect, useRef } from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../select'
import { FieldRendererProps } from './types'

export const SelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const selectOptions = field.typeOptions?.select?.options ?? []
  const hasSetDefault = useRef(false)

  // Set initial value on first render if no value is present but there's a default
  useEffect(() => {
    if (!hasSetDefault.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string
      if (defaultVal && defaultVal !== '') {
        onChange(defaultVal)
        hasSetDefault.current = true
      }
    }
  }, [value, field.defaultValue, onChange])

  return (
    <Select
      value={(value as string) ?? (field.defaultValue as string) ?? ''}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className={`w-full ${hasError ? 'border-red-500 border-2' : ''}`}>
        <SelectValue placeholder={`Select ${field.label || field.name}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        {selectOptions.map((opt) => (
          <SelectItem key={opt.value} value={opt.value ?? ''}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
