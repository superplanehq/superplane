import { Link } from "../Link/link";
import { twMerge } from "tailwind-merge";

export function Text({ className, ...props }: React.ComponentPropsWithoutRef<"p">) {
  return (
    <p
      data-slot="text"
      {...props}
      className={twMerge("text-base/6 text-gray-500 sm:text-sm/6 dark:text-gray-400", className)}
    />
  );
}

export function TextLink({ className, ...props }: React.ComponentPropsWithoutRef<typeof Link>) {
  return (
    <Link
      {...props}
      className={twMerge(
        "text-gray-950 underline decoration-gray-950/50 data-hover:decoration-gray-950 dark:text-white dark:decoration-white/50 dark:data-hover:decoration-white",
        className,
      )}
    />
  );
}

export function Strong({ className, ...props }: React.ComponentPropsWithoutRef<"strong">) {
  return <strong {...props} className={twMerge("font-medium text-gray-950 dark:text-white", className)} />;
}

export function Code({ className, ...props }: React.ComponentPropsWithoutRef<"code">) {
  return (
    <code
      {...props}
      className={twMerge(
        "rounded-sm border border-gray-950/10 bg-gray-950/2.5 px-0.5 text-sm font-medium text-gray-950 sm:text-[0.8125rem] dark:border-white/20 dark:bg-white/5 dark:text-white",
        className,
      )}
    />
  );
}
