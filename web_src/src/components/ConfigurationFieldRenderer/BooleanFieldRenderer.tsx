import React from 'react'
import { FieldRendererProps } from './types'

export const BooleanFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  return (
    <input
      type="checkbox"
      checked={value ?? (field.defaultValue === 'true') ?? false}
      onChange={(e) => onChange(e.target.checked)}
      className="h-4 w-4 rounded border-gray-300 dark:border-zinc-700"
    />
  )
}
