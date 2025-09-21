import { useCallback, useEffect } from 'react';
import { EventSourceSchedule, EventSourceScheduleType, ScheduleWeekDay } from '@/api-client/types.gen';
import { ValidationField } from './shared/ValidationField';

interface ScheduleConfigurationProps {
  schedule?: EventSourceSchedule | null;
  onScheduleChange: (schedule: EventSourceSchedule | null) => void;
  errors?: Record<string, string>;
}

const SCHEDULE_TYPE_OPTIONS: { value: EventSourceScheduleType; label: string }[] = [
  { value: 'TYPE_HOURLY', label: 'Hourly' },
  { value: 'TYPE_DAILY', label: 'Daily' },
  { value: 'TYPE_WEEKLY', label: 'Weekly' },
];

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

    if (type === 'TYPE_HOURLY') {
      newSchedule.hourly = {
        minute: schedule?.hourly?.minute ?? 0
      };
    } else if (type === 'TYPE_DAILY') {
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

  const handleMinuteChange = useCallback((minute: number) => {
    if (!schedule || schedule.type !== 'TYPE_HOURLY') return;

    const newSchedule: EventSourceSchedule = {
      ...schedule,
      hourly: { minute }
    };

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
            <select
              value={schedule?.type || 'TYPE_DAILY'}
              onChange={(e) => handleScheduleTypeChange(e.target.value as EventSourceScheduleType)}
              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${
                errors.scheduleType
                  ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                  : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
              }`}
            >
              {SCHEDULE_TYPE_OPTIONS.map(({ value, label }) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </ValidationField>

          {/* Hourly Minute Selection */}
          {schedule?.type === 'TYPE_HOURLY' && (
            <ValidationField
              label="Minute"
              error={errors.minute}
              required={true}
            >
              <select
                value={schedule.hourly?.minute ?? 0}
                onChange={(e) => handleMinuteChange(parseInt(e.target.value))}
                className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${
                  errors.minute
                    ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                    : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                }`}
              >
                {Array.from({ length: 12 }, (_, i) => i * 5).map((minute) => (
                  <option key={minute} value={minute}>
                    :{minute.toString().padStart(2, '0')}
                  </option>
                ))}
              </select>
            </ValidationField>
          )}

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
          {(schedule?.type === 'TYPE_DAILY' || schedule?.type === 'TYPE_WEEKLY') && (
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
          )}
    </div>
  );
}