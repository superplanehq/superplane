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
        "flex flex-col border-b border-edge-subtle p-4 [&>[data-slot=section]+[data-slot=section]]:mt-2.5",
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
        "flex flex-col border-t border-edge-subtle p-4 [&>[data-slot=section]+[data-slot=section]]:mt-2.5",
      )}
    />
  );
}

export function SidebarSection({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div {...props} data-slot="section" className={clsx(className, "flex flex-col")} />;
}

export function SidebarDivider({ className, ...props }: React.ComponentPropsWithoutRef<"hr">) {
  return <hr {...props} className={clsx(className, "my-4 border-t border-edge-subtle lg:-mx-4")} />;
}

export function SidebarSpacer({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div aria-hidden="true" {...props} className={clsx(className, "mt-8 flex-1")} />;
}

export function SidebarHeading({ className, ...props }: React.ComponentPropsWithoutRef<"h3">) {
  return <h3 {...props} className={clsx(className, "mb-1 px-2 text-xs/6 font-medium text-content-secondary")} />;
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
    "flex w-full items-center gap-3 rounded-lg px-2 py-2.5 text-left text-base/6 font-medium text-content-primary sm:py-2 sm:text-sm/5",
    "hover:bg-action-neutral-hover hover:text-content-primary",
    "active:bg-action-neutral-hover active:text-content-primary",
    current && "bg-action-neutral text-content-primary",
  );

  return (
    <span className={clsx(className, "relative")}>
      {current && <span className="absolute inset-y-2 -left-4 w-0.5 rounded-full bg-content-primary" />}
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
