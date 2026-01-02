"use client";

import clsx from "clsx";
import React from "react";

export function Sidebar({ className, ...props }: React.ComponentPropsWithoutRef<"nav">) {
  return <nav {...props} className={clsx(className, "flex h-full min-h-0 flex-col")} />;
}

export function SidebarHeader({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return (
    <div
      {...props}
      className={clsx(
        className,
        "flex flex-col border-b border-gray-950/5 p-4 dark:border-white/5 [&>[data-slot=section]+[data-slot=section]]:mt-2.5",
      )}
    />
  );
}

export function SidebarBody({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div {...props} className={clsx(className, "flex flex-1 flex-col overflow-y-auto")} />;
}

export function SidebarFooter({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return (
    <div
      {...props}
      className={clsx(
        className,
        "flex flex-col border-t border-gray-950/5 p-4 dark:border-white/5 [&>[data-slot=section]+[data-slot=section]]:mt-2.5",
      )}
    />
  );
}

export function SidebarSection({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div {...props} data-slot="section" className={clsx(className, "flex flex-col")} />;
}

export function SidebarDivider({ className, ...props }: React.ComponentPropsWithoutRef<"hr">) {
  return <hr {...props} className={clsx(className, "my-4 border-t border-gray-950/5 lg:-mx-4 dark:border-white/5")} />;
}

export function SidebarSpacer({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div aria-hidden="true" {...props} className={clsx(className, "mt-8 flex-1")} />;
}

export function SidebarHeading({ className, ...props }: React.ComponentPropsWithoutRef<"h3">) {
  return (
    <h3 {...props} className={clsx(className, "mb-1 px-2 text-xs/6 font-medium text-gray-500 dark:text-gray-400")} />
  );
}

export function SidebarItem({
  current,
  className,
  children,
  onClick,
  ...props
}: {
  current?: boolean;
  className?: string;
  children: React.ReactNode;
  onClick?: () => void;
} & React.ComponentPropsWithoutRef<"button">) {
  const classes = clsx(
    "flex w-full items-center gap-3 rounded-lg px-2 py-2.5 text-left text-base/6 font-medium text-gray-950 sm:py-2 sm:text-sm/5",
    "hover:bg-gray-950/5 hover:text-gray-950",
    "active:bg-gray-950/5 active:text-gray-950",
    "dark:text-white",
    "dark:hover:bg-white/5 dark:hover:text-white",
    "dark:active:bg-white/5 dark:active:text-white",
    current && "bg-gray-950/5 text-gray-950 dark:bg-white/5 dark:text-white",
  );

  return (
    <span className={clsx(className, "relative")}>
      {current && <span className="absolute inset-y-2 -left-4 w-0.5 rounded-full bg-gray-950 dark:bg-white" />}
      <button
        {...props}
        onClick={onClick}
        className={clsx("cursor-pointer", classes)}
        data-current={current ? "true" : undefined}
      >
        {children}
      </button>
    </span>
  );
}

export function SidebarLabel({ className, ...props }: React.ComponentPropsWithoutRef<"span">) {
  return <span {...props} className={clsx(className, "truncate")} />;
}
