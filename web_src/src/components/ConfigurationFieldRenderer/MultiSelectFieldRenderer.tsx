import React from 'react'
import { FieldRendererProps } from './types'
import { MultiCombobox, MultiComboboxLabel } from '../MultiCombobox/multi-combobox'

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

  // Get current selected values
  const currentValue = value ?? (field.defaultValue ? JSON.parse(field.defaultValue) : [])

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
