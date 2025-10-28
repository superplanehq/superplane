import React from 'react'
import { Input } from '../input'
import { FieldRendererProps } from './types'

export const TimeFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  return (
    <Input
      type="time"
      value={value ?? field.defaultValue ?? ''}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.typeOptions?.time?.format || 'HH:MM'}
    />
  )
}
