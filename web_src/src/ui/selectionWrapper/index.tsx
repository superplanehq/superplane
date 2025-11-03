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
  const baseClasses = fullRounded ? "rounded-full" : "rounded-lg";
  const selectedClasses = selected ? " ring-[6px] ring-blue-200 ring-offset-0" : "";

  return <div className={`${baseClasses}${selectedClasses}`}>{children}</div>;
};
