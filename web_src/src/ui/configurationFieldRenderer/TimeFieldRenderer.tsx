import React from 'react'
import { Input } from '../input'
import { FieldRendererProps } from './types'

export const TimeFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <Input
      type="time"
      value={(value as string) ?? (field.defaultValue as string) ?? ''}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.typeOptions?.time?.format || 'HH:MM'}
      className={hasError ? 'border-red-500 border-2' : ''}
    />
  )
}
