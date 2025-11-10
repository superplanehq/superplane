import React, { useState, useCallback } from "react";
import clsx from "clsx";
import Tippy from "@tippyjs/react";
import "tippy.js/dist/tippy.css";

export interface Tab {
  id: string;
  label: string;
  icon?: React.ReactNode;
  count?: number;
  disabled?: boolean;
  disabledTooltip?: string;
}

export interface TabsProps {
  tabs: Tab[];
  defaultTab?: string;
  onTabChange?: (tabId: string) => void;
  className?: string;
  variant?: "default" | "pills" | "underline" | "dark-underline";
  buttonClasses?: string;
}

export interface TabsState {
  active: string;
  tabs: Tab[];
}

export function useTabs(defaultTab: string, tabs: Tab[]): TabsState & { setActive: (tabId: string) => void } {
  const [active, setActive] = useState(defaultTab);

  const setActiveTab = useCallback((tabId: string) => {
    setActive(tabId);
  }, []);

  return {
    active,
    tabs,
    setActive: setActiveTab,
  };
}

export function Tabs({ tabs, defaultTab, onTabChange, className, variant = "default", buttonClasses }: TabsProps) {
  const [activeTab, setActiveTab] = useState(defaultTab || tabs[0]?.id || "");

  const handleTabClick = useCallback(
    (tabId: string) => {
      setActiveTab(tabId);
      onTabChange?.(tabId);
    },
    [onTabChange],
  );

  const baseClasses = clsx(
    "w-full",
    {
      "border-b border-zinc-200 dark:border-zinc-700":
        variant === "default" || variant === "underline" || variant === "dark-underline",
    },
    className,
  );

  const navClasses = clsx("flex", {
    "gap-1 p-1 bg-zinc-100 dark:bg-zinc-800 rounded-lg": variant === "pills",
    "gap-6": variant === "default",
    "gap-4": variant === "underline" || variant === "dark-underline",
  });

  return (
    <div className={baseClasses}>
      <nav className={navClasses}>
        {tabs.map((tab) => (
          <TabItem
            key={tab.id}
            tab={tab}
            activeTab={activeTab}
            onClick={handleTabClick}
            variant={variant}
            buttonClasses={buttonClasses}
          />
        ))}
      </nav>
    </div>
  );
}

function TabItem({
  tab,
  activeTab,
  onClick,
  variant,
  buttonClasses: additionalButtonClasses,
}: {
  tab: Tab;
  activeTab: string;
  onClick: (tabId: string) => void;
  variant: "default" | "pills" | "underline" | "dark-underline";
  buttonClasses?: string;
}) {
  const isActive = activeTab === tab.id;
  const isDisabled = tab.disabled;

  const handleClick = useCallback(() => {
    if (!isDisabled) {
      onClick(tab.id);
    }
  }, [tab.id, onClick, isDisabled]);

  const buttonClasses = clsx(
    "relative flex items-center gap-2 font-medium text-sm transition-all duration-200 ease-in-out focus:outline-hidden",
    additionalButtonClasses,
    {
      "px-2 py-3 border-b-2 border-transparent": variant === "default",
      "text-blue-600 border-blue-500 dark:text-blue-400": variant === "default" && isActive,

      "px-3 py-2 rounded-md": variant === "pills",
      "bg-white text-zinc-900 shadow-sm dark:bg-zinc-700 dark:text-white": variant === "pills" && isActive,
      "text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white":
        variant === "pills" && !isActive && !isDisabled,

      "px-3 py-3 relative": variant === "underline" || variant === "dark-underline",
      "text-blue-600 dark:text-blue-400": variant === "underline" && isActive,
      "text-blue-600 dark:text-blue-500": variant === "dark-underline" && isActive,
      "bg-zinc-300 dark:bg-zinc-900": variant === "dark-underline" && isActive,

      "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300":
        (variant === "default" || variant === "underline" || variant === "dark-underline") && !isActive && !isDisabled,

      "opacity-50 cursor-not-allowed": isDisabled,
      "cursor-pointer": !isDisabled,
    },
  );

  const tabButton = (
    <button
      type="button"
      className={buttonClasses}
      onClick={handleClick}
      disabled={isDisabled}
      data-testid={`tab-${tab.label.toLowerCase()}`}
    >
      {tab.icon && <span className="flex-shrink-0 w-4 h-4">{tab.icon}</span>}
      <span className="leading-none whitespace-nowrap text-center w-full">{tab.label}</span>
      {tab.count && tab.count > 0 && (
        <span className="inline-flex items-center justify-center text-xs font-medium rounded-full min-w-[1.25rem] h-5 px-1.5 bg-zinc-100 text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300">
          {tab.count > 99 ? "99+" : tab.count}
        </span>
      )}
      {variant === "underline" ||
        (variant === "dark-underline" && (
          <div
            className={clsx("absolute inset-x-0 bottom-0 h-0.5 bg-blue-500 transition-all duration-200 ease-in-out", {
              "scale-x-100": isActive,
              "scale-x-0": !isActive,
            })}
          />
        ))}
    </button>
  );

  // Wrap with Tippy if disabled and tooltip is provided
  if (isDisabled && tab.disabledTooltip) {
    return (
      <Tippy content={tab.disabledTooltip} placement="top">
        <div>{tabButton}</div>
      </Tippy>
    );
  }

  return tabButton;
}

export function ControlledTabs({
  tabs,
  activeTab,
  onTabChange,
  className,
  variant = "default",
  buttonClasses,
}: {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (tabId: string) => void;
  className?: string;
  variant?: "default" | "pills" | "underline" | "dark-underline";
  buttonClasses?: string;
}) {
  const handleTabClick = useCallback(
    (tabId: string) => {
      onTabChange(tabId);
    },
    [onTabChange],
  );

  const baseClasses = clsx(
    "w-full h-full",
    {
      "border-b border-zinc-200 dark:border-zinc-700": variant === "default" || variant === "underline",
    },
    className,
  );

  const navClasses = clsx("flex h-full", {
    "gap-1 p-1 bg-zinc-100 dark:bg-zinc-800 rounded-lg": variant === "pills",
    "gap-1": variant === "default",
    "gap-4": variant === "underline" || variant === "dark-underline",
  });

  return (
    <div className={baseClasses}>
      <nav className={navClasses}>
        {tabs.map((tab) => (
          <TabItem
            key={tab.id}
            tab={tab}
            activeTab={activeTab}
            onClick={handleTabClick}
            variant={variant}
            buttonClasses={buttonClasses}
          />
        ))}
      </nav>
    </div>
  );
}
