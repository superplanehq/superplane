import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive:
          "bg-destructive text-white hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/60",
        outline:
          "border border-slate-950/20 bg-background text-gray-800 shadow-xs hover:bg-accent hover:text-accent-foreground dark:bg-gray-800 dark:border-gray-600/70 dark:text-gray-100 dark:hover:bg-gray-700 dark:hover:text-gray-100",
        secondary:
          "bg-secondary text-gray-800 hover:bg-secondary/80 dark:bg-gray-800 dark:text-gray-100 dark:hover:bg-gray-700",
        ghost:
          "hover:bg-accent hover:text-accent-foreground dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:text-gray-100",
        link: "text-primary underline-offset-4 hover:underline",
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
