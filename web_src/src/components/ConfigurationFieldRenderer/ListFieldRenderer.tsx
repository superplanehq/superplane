import React from 'react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { FieldRendererProps } from './types'
import { ConfigurationFieldRenderer } from './index'

export const ListFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, domainId, domainType }) => {
  const items = Array.isArray(value) ? value : []
  const listOptions = field.typeOptions?.list
  const itemDefinition = listOptions?.itemDefinition

  const addItem = () => {
    const newItem = itemDefinition?.type === 'object'
      ? {}
      : itemDefinition?.type === 'number'
      ? 0
      : ''
    onChange([...items, newItem])
  }

  const removeItem = (index: number) => {
    const newItems = items.filter((_, i) => i !== index)
    onChange(newItems.length > 0 ? newItems : undefined)
  }

  const updateItem = (index: number, newValue: any) => {
    const newItems = [...items]
    newItems[index] = newValue
    onChange(newItems)
  }

  return (
    <div className="space-y-3">
      {items.map((item, index) => (
        <div key={index} className="flex gap-2 items-center">
          <div className="flex-1">
            {itemDefinition?.type === 'object' && itemDefinition.schema ? (
              <div className="border border-gray-300 dark:border-zinc-700 rounded-md p-4 space-y-4">
                {itemDefinition.schema.map((schemaField) => (
                  <ConfigurationFieldRenderer
                    key={schemaField.name}
                    field={schemaField}
                    value={item[schemaField.name!]}
                    onChange={(val) => {
                      const newItem = { ...item, [schemaField.name!]: val }
                      updateItem(index, newItem)
                    }}
                    allValues={item}
                    domainId={domainId}
                    domainType={domainType}
                  />
                ))}
              </div>
            ) : (
              <Input
                type={itemDefinition?.type === 'number' ? 'number' : 'text'}
                value={item ?? ''}
                onChange={(e) => {
                  const val = itemDefinition?.type === 'number'
                    ? (e.target.value === '' ? undefined : Number(e.target.value))
                    : e.target.value
                  updateItem(index, val)
                }}
              />
            )}
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => removeItem(index)}
            className="mt-1"
          >
            <MaterialSymbol name="delete" className="text-red-500" />
          </Button>
        </div>
      ))}
      <Button
        variant="outline"
        onClick={addItem}
        className="w-full mt-3"
      >
        <MaterialSymbol name="add" />
        Add Item
      </Button>
    </div>
  )
}
