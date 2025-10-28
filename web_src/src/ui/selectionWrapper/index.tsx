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
  if (selected) {
    return (
      <div className={"border-6 border-blue-200 bg-blue-200 " + (fullRounded ? "rounded-full" : "rounded-lg")}>
        {children}
      </div>
    );
  }

  return <>{children}</>;
};