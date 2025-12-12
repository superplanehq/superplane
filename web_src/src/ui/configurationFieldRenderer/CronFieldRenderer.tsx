import React from "react";
import { Input } from "../input";
import { FieldRendererProps } from "./types";

export const CronFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? "";

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    onChange(newValue || undefined);
  };

  return (
    <div className="space-y-2">
      <Input
        type="text"
        value={currentValue}
        onChange={handleChange}
        placeholder={field.placeholder || "0 30 14 * * MON-FRI"}
        className={hasError ? "border-red-500 border-2" : ""}
        spellCheck={false}
      />

      <div className="text-xs text-gray-500 dark:text-zinc-400">
        <p className="mb-1">Cron format: <code className="bg-gray-100 dark:bg-zinc-800 px-1 rounded">second minute hour day month dayofweek</code></p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-1 text-xs">
          <div>• <code>0 30 14 * * *</code> - Daily at 14:30</div>
          <div>• <code>0 0 9 * * MON-FRI</code> - Weekdays at 9:00</div>
          <div>• <code>0 0 0 1 * *</code> - First day of month</div>
          <div>• <code>0 */15 * * * *</code> - Every 15 minutes</div>
        </div>
        <p className="mt-1">Valid wildcards: <code className="bg-gray-100 dark:bg-zinc-800 px-1 rounded">* , - /</code></p>
      </div>
    </div>
  );
};