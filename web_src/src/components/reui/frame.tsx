import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

/**
 * ReUI Frame. Frame sets --frame-panel-bg and --frame-panel-border-color;
 * FramePanel reads them as bg and border. Dense mode pulls the panel flush so
 * a header can sit on the outer frame above a nested input panel.
 */
const frameVariants = cva(
  [
    "relative flex flex-col bg-muted/50 gap-(--frame-gap) px-(--frame-px) py-(--frame-py) rounded-(--frame-radius)",
    "[--frame-radius:var(--radius-xl)]",
    "[--frame-gap:--spacing(0.75)] [--frame-px:--spacing(0.75)] [--frame-py:--spacing(0.75)] [--frame-panel-header-gap:0rem] [--frame-panel-footer-gap:--spacing(1)]",
    "[--frame-panel-px-adjust:0px] [--frame-panel-py-adjust:0px] [--frame-panel-header-px-adjust:0px] [--frame-panel-header-py-adjust:0px] [--frame-panel-footer-px-adjust:0px] [--frame-panel-footer-py-adjust:0px]",
    "[--frame-panel-px:calc(var(--frame-panel-px-base)+var(--frame-panel-px-adjust))] [--frame-panel-py:calc(var(--frame-panel-py-base)+var(--frame-panel-py-adjust))] [--frame-panel-header-px:calc(var(--frame-panel-header-px-base)+var(--frame-panel-header-px-adjust))] [--frame-panel-header-py:calc(var(--frame-panel-header-py-base)+var(--frame-panel-header-py-adjust))] [--frame-panel-footer-px:calc(var(--frame-panel-footer-px-base)+var(--frame-panel-footer-px-adjust))] [--frame-panel-footer-py:calc(var(--frame-panel-footer-py-base)+var(--frame-panel-footer-py-adjust))]",
    "[--frame-panel-bg:var(--color-card)] [--frame-panel-border-color:var(--color-border)] [--frame-border-color:var(--color-border)]",
    "[--frame-panel-radius:calc(var(--frame-radius)-var(--frame-px)-1px)]",
  ],
  {
    variants: {
      variant: {
        default: "border border-[var(--frame-border-color)] bg-clip-padding",
        inverse:
          "[--frame-panel-bg:color-mix(in_oklch,var(--color-muted)_40%,transparent)] border border-[var(--frame-border-color)] bg-background bg-clip-padding",
        ghost: "[--frame-panel-radius:calc(var(--frame-radius)-var(--frame-px))]",
      },
      spacing: {
        xs: "[--frame-panel-px-base:--spacing(2)] [--frame-panel-py-base:--spacing(2)] [--frame-panel-header-px-base:--spacing(2)] [--frame-panel-header-py-base:--spacing(0.5)] [--frame-panel-footer-px-base:--spacing(2)] [--frame-panel-footer-py-base:--spacing(0.5)]",
        sm: "[--frame-panel-px-base:--spacing(3)] [--frame-panel-py-base:--spacing(3.5)] [--frame-panel-header-px-base:--spacing(3)] [--frame-panel-header-py-base:--spacing(1.5)] [--frame-panel-footer-px-base:--spacing(3)] [--frame-panel-footer-py-base:--spacing(1.5)]",
        default:
          "[--frame-panel-px-base:--spacing(4)] [--frame-panel-py-base:--spacing(4)] [--frame-panel-header-px-base:--spacing(4)] [--frame-panel-header-py-base:--spacing(2)] [--frame-panel-footer-px-base:--spacing(4)] [--frame-panel-footer-py-base:--spacing(2)]",
        lg: "[--frame-panel-px-base:--spacing(5)] [--frame-panel-py-base:--spacing(5)] [--frame-panel-header-px-base:--spacing(5)] [--frame-panel-header-py-base:--spacing(2.5)] [--frame-panel-footer-px-base:--spacing(5)] [--frame-panel-footer-py-base:--spacing(2.5)]",
      },
      stacked: {
        true: [
          "gap-0 *:has-[+[data-slot=frame-panel]]:rounded-b-none",
          "*:has-[+[data-slot=frame-panel]]:before:hidden",
          "*:[[data-slot=frame-panel]+[data-slot=frame-panel]]:rounded-t-none",
          "*:[[data-slot=frame-panel]+[data-slot=frame-panel]]:border-t-0",
        ],
        false: [
          "data-[spacing=sm]:*:[[data-slot=frame-panel]+[data-slot=frame-panel]]:mt-0.5",
          "data-[spacing=default]:*:[[data-slot=frame-panel]+[data-slot=frame-panel]]:mt-1",
          "data-[spacing=lg]:*:[[data-slot=frame-panel]+[data-slot=frame-panel]]:mt-2",
        ],
      },
      dense: {
        true: "p-0 gap-0 border-[var(--frame-border-color)] [--frame-panel-radius:var(--frame-radius)] [&_[data-slot=frame-panel]]:-mx-px [&_[data-slot=frame-panel]]:before:hidden [&_[data-slot=frame-panel]:last-child]:-mb-px [&:not(:has([data-slot=frame-panel-header]))_[data-slot=frame-panel]:is(:first-child)]:-mt-px",
        false: "",
      },
    },
    defaultVariants: {
      variant: "default",
      spacing: "default",
      stacked: false,
      dense: false,
    },
  },
);

function Frame({
  className,
  variant,
  spacing,
  stacked,
  dense,
  ...props
}: React.ComponentProps<"div"> & VariantProps<typeof frameVariants>) {
  return (
    <div
      className={cn(frameVariants({ variant, spacing, stacked, dense }), className)}
      data-slot="frame"
      data-spacing={spacing}
      {...props}
    />
  );
}

function FramePanel({
  className,
  fit,
  ...props
}: React.ComponentProps<"div"> & { fit?: boolean }) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-(--frame-panel-radius) border border-(--frame-panel-border-color) bg-(--frame-panel-bg) bg-clip-padding shadow-xs",
        !fit && "grow",
        "before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--frame-panel-radius)-1px)] before:shadow-black/5",
        "dark:bg-clip-border dark:before:shadow-white/5",
        "px-(--frame-panel-px) py-(--frame-panel-py)",
        className,
      )}
      data-slot="frame-panel"
      {...props}
    />
  );
}

function FrameHeader({ className, ...props }: React.ComponentProps<"header">) {
  return (
    <header
      className={cn(
        "flex flex-col gap-(--frame-panel-header-gap) px-(--frame-panel-header-px) py-(--frame-panel-header-py)",
        className,
      )}
      data-slot="frame-panel-header"
      {...props}
    />
  );
}

function FrameTitle({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("text-sm font-semibold", className)} data-slot="frame-panel-title" {...props} />;
}

function FrameDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("text-muted-foreground text-sm", className)} data-slot="frame-panel-description" {...props} />
  );
}

function FrameFooter({ className, ...props }: React.ComponentProps<"footer">) {
  return (
    <footer
      className={cn(
        "flex flex-col gap-(--frame-panel-footer-gap) px-(--frame-panel-footer-px) py-(--frame-panel-footer-py)",
        className,
      )}
      data-slot="frame-panel-footer"
      {...props}
    />
  );
}

export { Frame, FramePanel, FrameHeader, FrameTitle, FrameDescription, FrameFooter, frameVariants };
