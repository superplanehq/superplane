import React from "react";
import { Input } from "@/components/ui/input";
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
        placeholder={field.placeholder || "30 14 * * MON-FRI"}
        className=""
        spellCheck={false}
      />

      <div className="text-xs text-gray-500 dark:text-gray-400">
        <div className="space-y-1">
          <p className="font-medium">Wildcards:</p>
          <div className="ml-2 space-y-0.5">
            <div>
              <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">*</code> any value
            </div>
            <div>
              <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">,</code> value list separator
            </div>
            <div>
              <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">-</code> range of values
            </div>
            <div>
              <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">/</code> step values
            </div>
          </div>
          <p className="mt-2">
            Check{" "}
            <a
              href="https://crontab.guru"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 dark:text-blue-400 hover:underline"
            >
              Crontab Guru
            </a>{" "}
            for more details on cron expressions
          </p>
        </div>
      </div>
    </div>
  );
};
