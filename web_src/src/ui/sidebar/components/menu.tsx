import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { Skeleton } from "../../skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "../../tooltip";
import { cn } from "@/lib/utils";

const SidebarMenu = React.forwardRef<HTMLUListElement, React.ComponentProps<"ul">>(({ className, ...props }, ref) => (
  <ul ref={ref} data-sidebar="menu" className={cn("flex w-full min-w-0 flex-col gap-1", className)} {...props} />
));
SidebarMenu.displayName = "SidebarMenu";

const SidebarMenuItem = React.forwardRef<HTMLLIElement, React.ComponentProps<"li">>(({ className, ...props }, ref) => (
  <li ref={ref} data-sidebar="menu-item" className={cn("group/menu-item relative", className)} {...props} />
));
SidebarMenuItem.displayName = "SidebarMenuItem";

const sidebarMenuButtonVariants = cva(
  [
    "peer data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground [&>svg[data-slot=icon]]:text-sidebar-accent-foreground focus-visible:ring-sidebar-ring relative flex w-full items-center gap-2 overflow-hidden rounded-md px-2 py-1.5 text-left text-sm outline-none transition-[margin,opacity,padding] focus-visible:ring-2",
    "hover:bg-sidebar-accent/80 hover:text-sidebar-accent-foreground focus-visible:bg-sidebar-accent focus-visible:text-sidebar-accent-foreground",
    "disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50",
    "group-has-data-[menu=open]/menu-item:bg-sidebar-accent group-has-data-[menu=open]/menu-item:text-sidebar-accent-foreground",
    "[&>span[data-sidebar=menu-action]]:text-sidebar-accent-foreground [&>span[data-sidebar=menu-action]]:opacity-0 [&>span[data-sidebar=menu-action]]:transition-opacity [&>span[data-sidebar=menu-action]]:hover:opacity-100 [&>span[data-sidebar=menu-action]]:focus:opacity-100",
    "[&>span[data-sidebar=menu-badge]]:ml-auto [&>span[data-sidebar=menu-badge]]:transition-opacity",
    "[&>span[data-sidebar=menu-badge]]:group-has-data-[menu=open]/menu-item:opacity-0 [&>span[data-sidebar=menu-badge]]:group-data-[state=collapsed]/menu-item:opacity-0",
    "[&>svg[data-slot=icon]]:size-4 [&>svg[data-slot=icon]]:shrink-0 [&>svg[data-slot=icon]]:text-sidebar-foreground/70",
    "[&>[data-sidebar=menu-button]]:flex [&>[data-sidebar=menu-button]]:w-full [&>[data-sidebar=menu-button]]:items-center [&>[data-sidebar=menu-button]]:gap-2 [&>[data-sidebar=menu-button]]:text-left",
    "[&>[data-sidebar=menu-button]]:group-data-[state=collapsed]/menu-item:hidden [&>[data-sidebar=menu-button]]:group-data-[collapsible=icon]/menu-item:-ml-10 [&>[data-sidebar=menu-button]]:group-data-[collapsible=icon]/menu-item:w-0 [&>[data-sidebar=menu-button]]:group-data-[collapsible=icon]/menu-item:opacity-0",
  ].join(" "),
  {
    variants: {
      variant: {
        default: "",
        outline:
          "border bg-transparent data-[state=open]:bg-background data-[state=open]:text-sidebar-foreground data-[state=open]:shadow-sm",
      },
      size: {
        default: "h-8 text-sm [&>svg[data-slot=icon]]:size-4 [&>svg[data-slot=icon]]:shrink-0",
        sm: "h-7 text-xs [&>svg[data-slot=icon]]:size-3.5",
        lg: "h-12 text-base [&>svg[data-slot=icon]]:size-5",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

type SidebarMenuButtonProps = React.ComponentProps<"button"> &
  VariantProps<typeof sidebarMenuButtonVariants> & {
    tooltip?: string;
    isActive?: boolean;
    asChild?: boolean;
  };

const SidebarMenuButton = React.forwardRef<HTMLButtonElement, SidebarMenuButtonProps>(
  ({ className, variant, size, tooltip, isActive, asChild, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    const button = (
      <Comp
        ref={ref}
        data-sidebar="menu-button"
        data-size={size}
        data-active={isActive}
        className={cn(sidebarMenuButtonVariants({ variant, size }), className)}
        {...props}
      />
    );

    if (!tooltip) {
      return button;
    }

    return (
      <Tooltip delayDuration={0}>
        <TooltipTrigger asChild>{button}</TooltipTrigger>
        <TooltipContent side="right">{tooltip}</TooltipContent>
      </Tooltip>
    );
  },
);
SidebarMenuButton.displayName = "SidebarMenuButton";

const SidebarMenuAction = React.forwardRef<HTMLSpanElement, React.ComponentProps<"span">>(
  ({ className, ...props }, ref) => (
    <span
      ref={ref}
      data-sidebar="menu-action"
      className={cn(
        "flex items-center text-sm opacity-0 transition-opacity group-hover/menu-item:opacity-100 group-focus-visible/menu-item:opacity-100",
        className,
      )}
      {...props}
    />
  ),
);
SidebarMenuAction.displayName = "SidebarMenuAction";

const SidebarMenuBadge = React.forwardRef<HTMLSpanElement, React.ComponentProps<"span">>(
  ({ className, ...props }, ref) => (
    <span
      ref={ref}
      data-sidebar="menu-badge"
      className={cn(
        "bg-sidebar-primary text-sidebar-primary-foreground ml-auto inline-flex items-center rounded-full px-2 py-0.5 text-xs",
        "peer-data-[size=sm]/menu-button:text-[10px]",
        "group-data-[collapsible=icon]:hidden",
        className,
      )}
      {...props}
    />
  ),
);
SidebarMenuBadge.displayName = "SidebarMenuBadge";

const SidebarMenuSkeleton = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<"div"> & {
    showIcon?: boolean;
  }
>(({ className, showIcon = false, ...props }, ref) => {
  const width = React.useMemo(() => {
    return `${Math.floor(Math.random() * 40) + 50}%`;
  }, []);

  return (
    <div
      ref={ref}
      data-sidebar="menu-skeleton"
      className={cn("flex h-8 items-center gap-2 rounded-md px-2", className)}
      {...props}
    >
      {showIcon && <Skeleton className="size-4 rounded-md" data-sidebar="menu-skeleton-icon" />}
      <Skeleton
        className="h-4 max-w-[--skeleton-width] flex-1"
        data-sidebar="menu-skeleton-text"
        style={
          {
            "--skeleton-width": width,
          } as React.CSSProperties
        }
      />
    </div>
  );
});
SidebarMenuSkeleton.displayName = "SidebarMenuSkeleton";

const SidebarMenuSub = React.forwardRef<HTMLUListElement, React.ComponentProps<"ul">>(
  ({ className, ...props }, ref) => (
    <ul
      ref={ref}
      data-sidebar="menu-sub"
      className={cn(
        "border-sidebar-border mx-3.5 flex min-w-0 translate-x-px flex-col gap-1 border-l px-2.5 py-0.5",
        "group-data-[collapsible=icon]:hidden",
        className,
      )}
      {...props}
    />
  ),
);
SidebarMenuSub.displayName = "SidebarMenuSub";

const SidebarMenuSubItem = React.forwardRef<HTMLLIElement, React.ComponentProps<"li">>(({ ...props }, ref) => (
  <li ref={ref} {...props} />
));
SidebarMenuSubItem.displayName = "SidebarMenuSubItem";

const SidebarMenuSubButton = React.forwardRef<
  HTMLAnchorElement,
  React.ComponentProps<"a"> & {
    asChild?: boolean;
    size?: "sm" | "md";
    isActive?: boolean;
  }
>(({ asChild = false, size = "md", isActive, className, ...props }, ref) => {
  const Comp = asChild ? Slot : "a";

  return (
    <Comp
      ref={ref}
      data-sidebar="menu-sub-button"
      data-size={size}
      data-active={isActive}
      className={cn(
        "text-sidebar-foreground ring-sidebar-ring hover:bg-sidebar-accent hover:text-sidebar-accent-foreground active:bg-sidebar-accent active:text-sidebar-accent-foreground [&>svg]:text-sidebar-accent-foreground flex h-7 min-w-0 -translate-x-px items-center gap-2 overflow-hidden rounded-md px-2 outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&>span:last-child]:truncate [&>svg]:size-4 [&>svg]:shrink-0",
        "data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground",
        size === "sm" && "text-xs",
        size === "md" && "text-sm",
        "group-data-[collapsible=icon]:hidden",
        className,
      )}
      {...props}
    />
  );
});
SidebarMenuSubButton.displayName = "SidebarMenuSubButton";

export {
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
};
