import React from 'react'
import { Input } from '../input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../select'
import { FieldRendererProps } from './types'

const TIME_UNITS = [
  { value: 'seconds', label: 'Seconds', multiplier: 1 },
  { value: 'minutes', label: 'Minutes', multiplier: 60 },
  { value: 'hours', label: 'Hours', multiplier: 3600 }
] as const

export const TimeIntervalFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const [inputValue, setInputValue] = React.useState<string>('')
  const [unit, setUnit] = React.useState<string>('seconds')

  // Parse the initial value (which is always in seconds)
  React.useEffect(() => {
    if (value !== undefined && value !== null) {
      const seconds = Number(value)
      if (!isNaN(seconds)) {
        // Try to find the best unit to display (largest unit that results in whole numbers)
        const hourValue = seconds / 3600
        const minuteValue = seconds / 60

        if (hourValue >= 1 && Number.isInteger(hourValue)) {
          setInputValue(hourValue.toString())
          setUnit('hours')
        } else if (minuteValue >= 1 && Number.isInteger(minuteValue)) {
          setInputValue(minuteValue.toString())
          setUnit('minutes')
        } else {
          setInputValue(seconds.toString())
          setUnit('seconds')
        }
      }
    } else if (field.defaultValue) {
      const defaultSeconds = Number(field.defaultValue)
      if (!isNaN(defaultSeconds)) {
        setInputValue(defaultSeconds.toString())
        setUnit('seconds')
      }
    }
  }, [value, field.defaultValue])

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value
    setInputValue(newValue)

    if (newValue === '') {
      onChange(undefined)
      return
    }

    const numValue = Number(newValue)
    if (!isNaN(numValue)) {
      const selectedUnit = TIME_UNITS.find(u => u.value === unit)
      const seconds = numValue * (selectedUnit?.multiplier || 1)
      onChange(Math.abs(seconds).toString())
    }
  }

  const handleUnitChange = (newUnit: string) => {
    setUnit(newUnit)

    if (inputValue !== '') {
      const numValue = Number(inputValue)
      if (!isNaN(numValue)) {
        const selectedUnit = TIME_UNITS.find(u => u.value === newUnit)
        const seconds = numValue * (selectedUnit?.multiplier || 1)
        onChange(seconds.toString())
      }
    }
  }

  return (
    <div className="flex gap-2">
      <div className="flex-1">
        <Input
          type="number"
          value={inputValue}
          onChange={handleInputChange}
          placeholder={`Enter ${field.name}`}
          min={0}
          step="any"
          className={hasError ? 'border-red-500 border-2' : ''}
        />
      </div>
      <div className="w-24 h-full">
        <Select value={unit} onValueChange={handleUnitChange}>
          <SelectTrigger className={hasError ? 'border-red-500 border-2 h-full' : 'h-full'}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {TIME_UNITS.map((timeUnit) => (
              <SelectItem key={timeUnit.value} value={timeUnit.value}>
                {timeUnit.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}