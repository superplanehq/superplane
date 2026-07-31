import { Link } from "../Link/link";
import { twMerge } from "tailwind-merge";

export function Text({ className, ...props }: React.ComponentPropsWithoutRef<"p">) {
  return (
    <p
      data-slot="text"
      {...props}
      className={twMerge("text-base/6 text-content-secondary sm:text-sm/6", className)}
    />
  );
}

export function TextLink({ className, ...props }: React.ComponentPropsWithoutRef<typeof Link>) {
  return (
    <Link
      {...props}
      className={twMerge(
        "text-content-link underline decoration-content-link/50 data-hover:decoration-content-link-hover",
        className,
      )}
    />
  );
}

export function Strong({ className, ...props }: React.ComponentPropsWithoutRef<"strong">) {
  return <strong {...props} className={twMerge("font-medium text-content-primary", className)} />;
}

export function Code({ className, ...props }: React.ComponentPropsWithoutRef<"code">) {
  return (
    <code
      {...props}
      className={twMerge(
        "rounded-sm border border-edge-default bg-surface-subtle px-0.5 text-sm font-medium text-content-primary sm:text-[0.8125rem]",
        className,
      )}
    />
  );
}
