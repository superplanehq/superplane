import clsx from "clsx";
import type React from "react";

const sizes = {
  xs: "sm:max-w-xs",
  sm: "sm:max-w-sm",
  md: "sm:max-w-md",
  lg: "sm:max-w-lg",
  xl: "sm:max-w-xl",
  "2xl": "sm:max-w-2xl",
  "3xl": "sm:max-w-3xl",
  "4xl": "sm:max-w-4xl",
  "5xl": "sm:max-w-5xl",
};

export function Dialog({
  size = "lg",
  className,
  children,
  open,
  onClose,
  ...props
}: {
  size?: keyof typeof sizes;
  className?: string;
  children: React.ReactNode;
  open: boolean;
  onClose: () => void;
} & React.ComponentPropsWithoutRef<"div">) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="fixed inset-0 bg-gray-950/20 dark:bg-gray-950/50" onClick={onClose} />
      <div
        className={clsx(
          className,
          sizes[size],
          "relative w-full min-w-0 rounded-2xl bg-white p-8 shadow-lg dark:bg-gray-900",
          "overflow-y-auto max-h-[100vh]",
        )}
        {...props}
      >
        {children}
      </div>
    </div>
  );
}

export function DialogTitle({ className, ...props }: React.ComponentPropsWithoutRef<"h2">) {
  return (
    <h2
      {...props}
      className={clsx(className, "text-lg/6 font-semibold text-balance text-gray-800 sm:text-base/6 dark:text-white")}
    />
  );
}

export function DialogDescription({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div {...props} className={clsx(className, "mt-2 text-pretty text-gray-500 dark:text-gray-400")} />;
}

export function DialogBody({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div {...props} className={clsx(className, "mt-6")} />;
}

export function DialogActions({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return (
    <div
      {...props}
      className={clsx(
        className,
        "mt-8 flex flex-col items-center justify-start gap-3 *:w-full sm:flex-row sm:*:w-auto",
      )}
    />
  );
}
