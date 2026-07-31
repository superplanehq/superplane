import { cva } from "class-variance-authority";

export const badgeVariants = cva(
  "inline-flex items-center justify-center rounded-md border px-2 py-0.5 text-xs font-medium w-fit whitespace-nowrap shrink-0 [&>svg]:size-3 gap-1 [&>svg]:pointer-events-none focus-visible:border-focus-ring focus-visible:ring-focus-ring/50 focus-visible:ring-[3px] aria-invalid:ring-status-danger/20 aria-invalid:border-status-danger transition-[color,box-shadow] overflow-hidden",
  {
    variants: {
      variant: {
        default: "border-transparent bg-action-primary text-action-primary-content [a&]:hover:bg-action-primary-hover",
        secondary: "border-transparent bg-action-neutral text-content-primary [a&]:hover:bg-action-neutral-hover",
        destructive:
          "border-transparent bg-status-danger text-content-inverse [a&]:hover:bg-status-danger/90 focus-visible:ring-status-danger/20",
        outline: "text-content-primary [a&]:hover:bg-action-neutral-hover [a&]:hover:text-content-primary",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
