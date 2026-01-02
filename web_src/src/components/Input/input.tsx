import React, { forwardRef } from "react";

import { Input as UiInput } from "@/components/ui/input";
import { InputGroup as UiInputGroup } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";

export function InputGroup(props: React.ComponentProps<typeof UiInputGroup>) {
  return <UiInputGroup {...props} />;
}

const dateTypes = ["date", "datetime-local", "month", "time", "week"];
type DateType = (typeof dateTypes)[number];

export const Input = forwardRef(function Input(
  {
    className,
    type,
    ...props
  }: {
    className?: string;
    type?: "email" | "number" | "password" | "search" | "tel" | "text" | "url" | DateType;
  } & React.ComponentPropsWithoutRef<"input">,
  ref: React.ForwardedRef<HTMLInputElement>,
) {
  return (
    <UiInput
      ref={ref}
      type={type}
      className={cn(
        type &&
          dateTypes.includes(type) && [
            "[&::-webkit-datetime-edit-fields-wrapper]:p-0",
            "[&::-webkit-date-and-time-value]:min-h-[1.5em]",
            "[&::-webkit-datetime-edit]:inline-flex",
            "[&::-webkit-datetime-edit]:p-0",
            "[&::-webkit-datetime-edit-year-field]:p-0",
            "[&::-webkit-datetime-edit-month-field]:p-0",
            "[&::-webkit-datetime-edit-day-field]:p-0",
            "[&::-webkit-datetime-edit-hour-field]:p-0",
            "[&::-webkit-datetime-edit-minute-field]:p-0",
            "[&::-webkit-datetime-edit-second-field]:p-0",
            "[&::-webkit-datetime-edit-millisecond-field]:p-0",
            "[&::-webkit-datetime-edit-meridiem-field]:p-0",
          ],
        className,
      )}
      {...props}
    />
  );
});
