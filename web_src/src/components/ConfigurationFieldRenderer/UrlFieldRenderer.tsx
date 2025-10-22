import React from 'react'
import { Input } from '../ui/input'
import { FieldRendererProps } from './types'

export const UrlFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  return (
    <Input
      type="url"
      value={value ?? field.defaultValue ?? ''}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder="https://example.com"
    />
  )
}
