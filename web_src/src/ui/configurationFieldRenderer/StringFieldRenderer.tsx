import React from 'react'
import { Input } from '../input'
import { FieldRendererProps } from './types'

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <Input
      type="text"
      value={(value as string) ?? (field.defaultValue as string) ?? ''}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={`Enter ${field.name}`}
      className={hasError ? 'border-red-500 border-2' : ''}
    />
  )
}
