import React from "react";

export interface SelectionWrapperProps {
  selected?: boolean;
  fullRounded?: boolean;
  children: React.ReactNode;
}

export const SelectionWrapper: React.FC<SelectionWrapperProps> = ({
  selected = false,
  fullRounded = false,
  children,
}) => {
  const baseClasses = fullRounded ? "rounded-full" : "rounded-md";
  const selectedClasses = selected ? " ring-[3px] ring-sky-300 dark:ring-sky-500 ring-offset-4 dark:ring-offset-gray-800" : "";

  return <div className={`${baseClasses}${selectedClasses}`}>{children}</div>;
};
