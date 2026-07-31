import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-focus-ring focus-visible:ring-focus-ring/50 focus-visible:ring-[3px] aria-invalid:ring-status-danger/20 aria-invalid:border-status-danger",
  {
    variants: {
      variant: {
        default: "bg-action-primary text-action-primary-content hover:bg-action-primary-hover",
        destructive:
          "bg-status-danger text-content-inverse hover:bg-status-danger/90 focus-visible:ring-status-danger/20",
        outline:
          "border border-edge-strong bg-surface-raised text-content-primary shadow-xs hover:bg-action-neutral-hover hover:text-content-primary",
        secondary: "bg-action-neutral text-content-primary hover:bg-action-neutral-hover",
        ghost: "text-content-primary hover:bg-action-neutral-hover hover:text-content-primary",
        link: "text-content-link underline-offset-4 hover:underline",
      },
      size: {
        default:
          "h-8 px-4 py-1.5 has-[>svg:first-child:not(:last-child)]:pl-3 has-[>svg:last-child:not(:first-child)]:pr-3",
        xs: "h-6 rounded-full gap-1 px-3 py-0.5 text-xs has-[>svg:first-child:not(:last-child)]:pl-2.5 has-[>svg:last-child:not(:first-child)]:pr-2.5",
        sm: "h-7 rounded-full gap-1 px-3 py-1 text-[13px] has-[>svg:first-child:not(:last-child)]:pl-2.5 has-[>svg:last-child:not(:first-child)]:pr-2.5",
        lg: "h-10 rounded-full px-8 has-[>svg:first-child:not(:last-child)]:pl-6 has-[>svg:last-child:not(:first-child)]:pr-6",
        icon: "size-9",
        "icon-sm": "size-8",
        "icon-xs": "size-7 rounded-full",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);
