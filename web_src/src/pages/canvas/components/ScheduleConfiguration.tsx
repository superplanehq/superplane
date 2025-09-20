import { useCallback, useEffect } from 'react';
import { EventSourceSchedule, EventSourceScheduleType, ScheduleWeekDay } from '@/api-client/types.gen';
import { ValidationField } from './shared/ValidationField';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

interface ScheduleConfigurationProps {
  schedule?: EventSourceSchedule | null;
  onScheduleChange: (schedule: EventSourceSchedule | null) => void;
  errors?: Record<string, string>;
}

const WEEKDAY_OPTIONS: { value: ScheduleWeekDay; label: string }[] = [
  { value: 'WEEK_DAY_MONDAY', label: 'Monday' },
  { value: 'WEEK_DAY_TUESDAY', label: 'Tuesday' },
  { value: 'WEEK_DAY_WEDNESDAY', label: 'Wednesday' },
  { value: 'WEEK_DAY_THURSDAY', label: 'Thursday' },
  { value: 'WEEK_DAY_FRIDAY', label: 'Friday' },
  { value: 'WEEK_DAY_SATURDAY', label: 'Saturday' },
  { value: 'WEEK_DAY_SUNDAY', label: 'Sunday' },
];

export function ScheduleConfiguration({
  schedule,
  onScheduleChange,
  errors = {}
}: ScheduleConfigurationProps) {

  // Initialize with default daily schedule if no schedule exists
  useEffect(() => {
    if (!schedule) {
      onScheduleChange({
        type: 'TYPE_DAILY',
        daily: {
          time: '09:00'
        }
      });
    }
  }, [schedule, onScheduleChange]);

  const handleScheduleTypeChange = useCallback((type: EventSourceScheduleType) => {
    const newSchedule: EventSourceSchedule = {
      type
    };

    if (type === 'TYPE_DAILY') {
      newSchedule.daily = {
        time: schedule?.daily?.time || '09:00'
      };
    } else if (type === 'TYPE_WEEKLY') {
      newSchedule.weekly = {
        weekDay: schedule?.weekly?.weekDay || 'WEEK_DAY_MONDAY',
        time: schedule?.weekly?.time || '09:00'
      };
    }

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const handleTimeChange = useCallback((time: string) => {
    if (!schedule) return;

    const newSchedule: EventSourceSchedule = { ...schedule };

    if (schedule.type === 'TYPE_DAILY') {
      newSchedule.daily = { ...schedule.daily, time };
    } else if (schedule.type === 'TYPE_WEEKLY') {
      newSchedule.weekly = { ...schedule.weekly, time };
    }

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const handleWeekDayChange = useCallback((weekDay: ScheduleWeekDay) => {
    if (!schedule || schedule.type !== 'TYPE_WEEKLY') return;

    const newSchedule: EventSourceSchedule = {
      ...schedule,
      weekly: { ...schedule.weekly, weekDay }
    };

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const getCurrentTime = (): string => {
    if (schedule?.type === 'TYPE_DAILY') {
      return schedule.daily?.time || '09:00';
    } else if (schedule?.type === 'TYPE_WEEKLY') {
      return schedule.weekly?.time || '09:00';
    }
    return '09:00';
  };

  return (
    <div className="space-y-4">
          {/* Schedule Type */}
          <ValidationField
            label="Schedule Type"
            error={errors.scheduleType}
            required={true}
          >
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => handleScheduleTypeChange('TYPE_DAILY')}
                className={`px-3 py-2 text-sm rounded-md border transition-colors ${
                  schedule?.type === 'TYPE_DAILY'
                    ? 'bg-blue-50 border-blue-300 text-blue-900 dark:bg-blue-900/20 dark:border-blue-600 dark:text-blue-300'
                    : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-zinc-800 dark:border-zinc-600 dark:text-zinc-300 dark:hover:bg-zinc-700'
                }`}
              >
                <div className="flex items-center justify-center gap-2">
                  <MaterialSymbol name="today" size="sm" />
                  Daily
                </div>
              </button>
              <button
                type="button"
                onClick={() => handleScheduleTypeChange('TYPE_WEEKLY')}
                className={`px-3 py-2 text-sm rounded-md border transition-colors ${
                  schedule?.type === 'TYPE_WEEKLY'
                    ? 'bg-blue-50 border-blue-300 text-blue-900 dark:bg-blue-900/20 dark:border-blue-600 dark:text-blue-300'
                    : 'bg-white border-gray-300 text-gray-700 hover:bg-gray-50 dark:bg-zinc-800 dark:border-zinc-600 dark:text-zinc-300 dark:hover:bg-zinc-700'
                }`}
              >
                <div className="flex items-center justify-center gap-2">
                  <MaterialSymbol name="date_range" size="sm" />
                  Weekly
                </div>
              </button>
            </div>
          </ValidationField>

          {/* Weekly Day Selection */}
          {schedule?.type === 'TYPE_WEEKLY' && (
            <ValidationField
              label="Day of Week"
              error={errors.weekDay}
              required={true}
            >
              <select
                value={schedule.weekly?.weekDay || 'WEEK_DAY_MONDAY'}
                onChange={(e) => handleWeekDayChange(e.target.value as ScheduleWeekDay)}
                className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${
                  errors.weekDay
                    ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                    : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                }`}
              >
                {WEEKDAY_OPTIONS.map(({ value, label }) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </ValidationField>
          )}

          {/* Time Selection */}
          <ValidationField
            label="Time (UTC)"
            error={errors.time}
            required={true}
          >
            <input
              type="time"
              value={getCurrentTime()}
              onChange={(e) => handleTimeChange(e.target.value)}
              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${
                errors.time
                  ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                  : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
              }`}
            />
          </ValidationField>
    </div>
  );
}