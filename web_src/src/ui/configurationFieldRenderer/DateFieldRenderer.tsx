import React from 'react'
import { Input } from '../input'
import { FieldRendererProps } from './types'

export const DateFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  return (
    <Input
      type="date"
      value={value ?? field.defaultValue ?? ''}
      onChange={(e) => onChange(e.target.value || undefined)}
    />
  )
}
