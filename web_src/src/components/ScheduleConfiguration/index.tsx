import { useCallback, useEffect } from 'react';
import { SuperplaneEventSourceSchedule, EventSourceScheduleType, ScheduleWeekDay } from '@/api-client/types.gen';
import { ValidationField } from '../ValidationField';
import { Select } from '../Select';
import { convertLocalTimeToUTC, convertUTCToLocalTime, getUserTimezoneDisplay } from '@/utils/timezone';

interface ScheduleConfigurationProps {
  schedule?: SuperplaneEventSourceSchedule | null;
  onScheduleChange: (schedule: SuperplaneEventSourceSchedule | null) => void;
  errors?: Record<string, string>;
}

const SCHEDULE_TYPE_OPTIONS = [
  { value: 'TYPE_HOURLY', label: 'Hourly' },
  { value: 'TYPE_DAILY', label: 'Daily' },
  { value: 'TYPE_WEEKLY', label: 'Weekly' },
];

const WEEKDAY_OPTIONS = [
  { value: 'WEEK_DAY_MONDAY', label: 'Monday' },
  { value: 'WEEK_DAY_TUESDAY', label: 'Tuesday' },
  { value: 'WEEK_DAY_WEDNESDAY', label: 'Wednesday' },
  { value: 'WEEK_DAY_THURSDAY', label: 'Thursday' },
  { value: 'WEEK_DAY_FRIDAY', label: 'Friday' },
  { value: 'WEEK_DAY_SATURDAY', label: 'Saturday' },
  { value: 'WEEK_DAY_SUNDAY', label: 'Sunday' },
];

const MINUTE_OPTIONS = Array.from({ length: 12 }, (_, i) => ({
  value: (i * 5).toString(),
  label: `:${(i * 5).toString().padStart(2, '0')}`
}));

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
          time: convertLocalTimeToUTC('09:00')
        }
      });
    }
  }, [schedule, onScheduleChange]);

  const handleScheduleTypeChange = useCallback((type: string) => {
    const newSchedule: SuperplaneEventSourceSchedule = {
      type: type as EventSourceScheduleType
    };

    if (type === 'TYPE_HOURLY') {
      newSchedule.hourly = {
        minute: schedule?.hourly?.minute ?? 0
      };
    } else if (type === 'TYPE_DAILY') {
      newSchedule.daily = {
        time: schedule?.daily?.time || convertLocalTimeToUTC('09:00')
      };
    } else if (type === 'TYPE_WEEKLY') {
      newSchedule.weekly = {
        weekDay: schedule?.weekly?.weekDay || 'WEEK_DAY_MONDAY',
        time: schedule?.weekly?.time || convertLocalTimeToUTC('09:00')
      };
    }

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const handleMinuteChange = useCallback((minute: string) => {
    if (!schedule || schedule.type !== 'TYPE_HOURLY') return;

    const newSchedule: SuperplaneEventSourceSchedule = {
      ...schedule,
      hourly: { minute: parseInt(minute) }
    };

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const handleTimeChange = useCallback((time: string) => {
    if (!schedule) return;

    const utcTime = convertLocalTimeToUTC(time);
    const newSchedule: SuperplaneEventSourceSchedule = { ...schedule };

    if (schedule.type === 'TYPE_DAILY') {
      newSchedule.daily = { ...schedule.daily, time: utcTime };
    } else if (schedule.type === 'TYPE_WEEKLY') {
      newSchedule.weekly = { ...schedule.weekly, time: utcTime };
    }

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const handleWeekDayChange = useCallback((weekDay: string) => {
    if (!schedule || schedule.type !== 'TYPE_WEEKLY') return;

    const newSchedule: SuperplaneEventSourceSchedule = {
      ...schedule,
      weekly: { ...schedule.weekly, weekDay: weekDay as ScheduleWeekDay }
    };

    onScheduleChange(newSchedule);
  }, [schedule, onScheduleChange]);

  const getCurrentTime = (): string => {
    if (schedule?.type === 'TYPE_DAILY') {
      const utcTime = schedule.daily?.time || convertLocalTimeToUTC('09:00');
      return convertUTCToLocalTime(utcTime);
    } else if (schedule?.type === 'TYPE_WEEKLY') {
      const utcTime = schedule.weekly?.time || convertLocalTimeToUTC('09:00');
      return convertUTCToLocalTime(utcTime);
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
            <Select
              options={SCHEDULE_TYPE_OPTIONS}
              value={schedule?.type || 'TYPE_DAILY'}
              onChange={handleScheduleTypeChange}
              error={!!errors.scheduleType}
            />
          </ValidationField>

          {/* Hourly Minute Selection */}
          {schedule?.type === 'TYPE_HOURLY' && (
            <ValidationField
              label="Minute"
              error={errors.minute}
              required={true}
            >
              <Select
                options={MINUTE_OPTIONS}
                value={(schedule.hourly?.minute ?? 0).toString()}
                onChange={handleMinuteChange}
                error={!!errors.minute}
              />
            </ValidationField>
          )}

          {/* Weekly Day Selection */}
          {schedule?.type === 'TYPE_WEEKLY' && (
            <ValidationField
              label="Day of Week"
              error={errors.weekDay}
              required={true}
            >
              <Select
                options={WEEKDAY_OPTIONS}
                value={schedule.weekly?.weekDay || 'WEEK_DAY_MONDAY'}
                onChange={handleWeekDayChange}
                error={!!errors.weekDay}
              />
            </ValidationField>
          )}

          {/* Time Selection */}
          {(schedule?.type === 'TYPE_DAILY' || schedule?.type === 'TYPE_WEEKLY') && (
            <ValidationField
              label={`Time (${getUserTimezoneDisplay()})`}
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