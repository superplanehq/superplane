import React from 'react'
import { Input } from '../ui/input'
import { FieldRendererProps } from './types'

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
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
}
