import clsx from "clsx";
import type React from "react";

export function Fieldset({ className, ...props }: { className?: string } & React.ComponentPropsWithoutRef<"fieldset">) {
  return <fieldset {...props} className={clsx(className, "*:data-[slot=text]:mt-1 [&>*+[data-slot=control]]:mt-6")} />;
}

export function Legend({ className, ...props }: { className?: string } & React.ComponentPropsWithoutRef<"legend">) {
  return (
    <legend
      data-slot="legend"
      {...props}
      className={clsx(
        className,
        "text-base/6 font-semibold text-gray-800data-disabled:opacity-50 sm:text-sm/6 dark:text-white",
      )}
    />
  );
}

export function FieldGroup({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return <div data-slot="control" {...props} className={clsx(className, "space-y-8")} />;
}

export function Field({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return (
    <div
      {...props}
      className={clsx(
        className,
        "[&>[data-slot=label]+[data-slot=control]]:mt-3",
        "[&>[data-slot=label]+[data-slot=description]]:mt-1",
        "[&>[data-slot=description]+[data-slot=control]]:mt-3",
        "[&>[data-slot=control]+[data-slot=description]]:mt-3",
        "[&>[data-slot=control]+[data-slot=error]]:mt-3",
        "*:data-[slot=label]:font-medium",
      )}
    />
  );
}

export function Label({ className, ...props }: React.ComponentPropsWithoutRef<"label">) {
  return (
    <label
      data-slot="label"
      {...props}
      className={clsx(
        "flex items-center gap-2 text-sm leading-none font-medium select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50",
        className,
      )}
    />
  );
}

export function Description({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return (
    <div
      data-slot="description"
      {...props}
      className={clsx(className, "text-base/6 text-gray-500 data-disabled:opacity-50 sm:text-sm/6 dark:text-gray-400")}
    />
  );
}

export function ErrorMessage({ className, ...props }: React.ComponentPropsWithoutRef<"div">) {
  return (
    <div
      {...props}
      className={clsx(className, "ml-1 text-xs absolute text-red-600 data-disabled:opacity-50 dark:text-red-500")}
    />
  );
}
