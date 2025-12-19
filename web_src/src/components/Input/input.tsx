import { twMerge } from "tailwind-merge";
import React, { forwardRef } from "react";

export function InputGroup({ children }: React.ComponentPropsWithoutRef<"span">) {
  return (
    <span
      data-slot="control"
      className={twMerge(
        "relative isolate block",
        "has-[[data-slot=icon]:first-child]:[&_input]:pl-10 has-[[data-slot=icon]:last-child]:[&_input]:pr-10 sm:has-[[data-slot=icon]:first-child]:[&_input]:pl-8 sm:has-[[data-slot=icon]:last-child]:[&_input]:pr-8",
        "*:data-[slot=icon]:pointer-events-none *:data-[slot=icon]:absolute *:data-[slot=icon]:top-3 *:data-[slot=icon]:z-10 *:data-[slot=icon]:size-5 sm:*:data-[slot=icon]:top-2.5 sm:*:data-[slot=icon]:size-4",
        "[&>[data-slot=icon]:first-child]:left-3 sm:[&>[data-slot=icon]:first-child]:left-2.5 [&>[data-slot=icon]:last-child]:right-3 sm:[&>[data-slot=icon]:last-child]:right-2.5",
        "*:data-[slot=icon]:text-gray-500 dark:*:data-[slot=icon]:text-gray-400",
      )}
    >
      {children}
    </span>
  );
}

const dateTypes = ["date", "datetime-local", "month", "time", "week"];
type DateType = (typeof dateTypes)[number];

export const Input = forwardRef(function Input(
  {
    className,
    ...props
  }: {
    className?: string;
    type?: "email" | "number" | "password" | "search" | "tel" | "text" | "url" | DateType;
  } & React.ComponentPropsWithoutRef<"input">,
  ref: React.ForwardedRef<HTMLInputElement>,
) {
  return (
    <span
      data-slot="control"
      className={twMerge([
        "relative block w-full",
        "before:absolute before:inset-px before:rounded-[calc(var(--radius-lg)-1px)] before:bg-white before:shadow-sm",
        "dark:before:hidden",
        "after:pointer-events-none after:absolute after:inset-0 after:rounded-lg after:ring-transparent after:ring-inset sm:focus-within:after:ring-2 sm:focus-within:after:ring-blue-500",
        "has-data-disabled:opacity-50 has-data-disabled:before:bg-gray-950/5 has-data-disabled:before:shadow-none",
        "has-data-invalid:before:shadow-red-500/10",
        className,
      ])}
    >
      <input
        ref={ref}
        {...props}
        className={twMerge([
          props.type &&
            dateTypes.includes(props.type) && [
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
          "relative block w-full appearance-none rounded-lg px-3 py-2 sm:px-3 sm:py-1.5",
          "text-base/6 text-gray-950 placeholder:text-gray-500 sm:text-sm/6 dark:text-white",
          "border border-gray-950/10 hover:border-gray-950/20 dark:border-white/10 dark:hover:border-white/20",
          "bg-transparent dark:bg-white/5",
          "focus:outline-none",
          "invalid:border-red-500 dark:invalid:border-red-500",
          "disabled:border-gray-950/20 dark:disabled:border-white/15 dark:disabled:bg-white/2.5",
          className,
        ])}
      />
    </span>
  );
});
