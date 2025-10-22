import React from 'react'
import { FieldRendererProps } from './types'

export const MultiSelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const multiSelectOptions = field.typeOptions?.multiSelect?.options ?? []

  return (
    <select
      multiple
      value={value ?? (field.defaultValue ? JSON.parse(field.defaultValue) : [])}
      onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
        const selected = Array.from(e.target.selectedOptions, opt => opt.value)
        onChange(selected.length > 0 ? selected : undefined)
      }}
      className="w-full px-3 py-2 border border-gray-300 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100"
      size={Math.min(multiSelectOptions.length ?? 5, 5)}
    >
      {multiSelectOptions.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  )
}
