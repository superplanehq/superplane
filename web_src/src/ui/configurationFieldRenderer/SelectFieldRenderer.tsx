import React from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../select'
import { FieldRendererProps } from './types'

export const SelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
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
}
