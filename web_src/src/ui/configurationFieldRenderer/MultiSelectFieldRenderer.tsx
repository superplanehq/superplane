import React, { useEffect } from 'react'
import { FieldRendererProps } from './types'
import { MultiCombobox, MultiComboboxLabel } from '@/components/MultiCombobox/multi-combobox'

interface SelectOption {
  id: string
  label: string
  value: string
}

export const MultiSelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const multiSelectOptions = field.typeOptions?.multiSelect?.options ?? []

  // Convert options to the format expected by MultiCombobox
  const comboboxOptions: SelectOption[] = multiSelectOptions.map((opt) => ({
    id: opt.value!,
    label: opt.label!,
    value: opt.value!,
  }))

  // Set initial value on first render if no value is present but there's a default
  useEffect(() => {
    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Array.isArray(field.defaultValue) ? field.defaultValue : (field.defaultValue ? JSON.parse(field.defaultValue as string) : [])
      if (Array.isArray(defaultVal) && defaultVal.length > 0) {
        onChange(defaultVal)
      }
    }
  }, [value, field.defaultValue, onChange])

  // Get current selected values
  const currentValue = value ?? (field.defaultValue ? (Array.isArray(field.defaultValue) ? field.defaultValue : JSON.parse(field.defaultValue as string)) : [])

  // Convert selected values to SelectOption objects
  const selectedOptions: SelectOption[] = currentValue.map((val: string) => {
    const option = comboboxOptions.find(opt => opt.value === val)
    return option || { id: val, label: val, value: val }
  })

  const handleChange = (selectedOptions: SelectOption[]) => {
    const selectedValues = selectedOptions.map(opt => opt.value)
    onChange(selectedValues.length > 0 ? selectedValues : undefined)
  }

  return (
    <MultiCombobox<SelectOption>
      options={comboboxOptions}
      displayValue={(option) => option.label}
      placeholder={`Select ${field.label || field.name}...`}
      value={selectedOptions}
      onChange={handleChange}
      showButton={true}
    >
      {(option) => (
        <MultiComboboxLabel>{option.label}</MultiComboboxLabel>
      )}
    </MultiCombobox>
  )
}
