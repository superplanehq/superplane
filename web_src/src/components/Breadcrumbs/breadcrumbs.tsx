import React from "react";
import { Link } from "../Link/link";
import { Icon } from "../Icon";
import clsx from "clsx";

export interface BreadcrumbItem {
  label: string;
  href?: string;
  icon?: string;
  current?: boolean;
  onClick?: () => void;
}

export interface BreadcrumbsProps {
  items: BreadcrumbItem[];
  className?: string;
  separator?: "/" | ">" | "•";
  showDivider?: boolean;
}

export function Breadcrumbs({ items, className, separator = "/", showDivider = true }: BreadcrumbsProps) {
  if (!items.length) return null;

  return (
    <nav className={clsx("flex items-center space-x-2 text-sm", className)} aria-label="Breadcrumb">
      {/* Divider line */}
      {showDivider && <div className="h-5 w-px bg-gray-300 dark:bg-gray-600 mr-4" />}

      {items.map((item, index) => (
        <React.Fragment key={index}>
          <div className="flex items-center">
            {item.current ? (
              // Current page (not clickable)
              <span className="text-gray-800 dark:text-gray-100 font-medium flex items-center" aria-current="page">
                {item.icon && <Icon name={item.icon} className="text-gray-700 dark:text-gray-300 mr-1" size="sm" />}
                {item.label}
              </span>
            ) : item.href ? (
              // Clickable link
              <Link
                href={item.href}
                className="text-gray-500 hover:text-gray-800 transition-colors flex items-center"
              >
                {item.icon && <Icon name={item.icon} className="text-blue-700 dark:text-blue-400 mr-1" size="sm" />}
                {item.label}
              </Link>
            ) : item.onClick ? (
              // Clickable button
              <button
                onClick={item.onClick}
                className="text-gray-500 hover:text-gray-800 transition-colors flex items-center"
              >
                {item.icon && <Icon name={item.icon} className="text-blue-700 dark:text-blue-400 mr-1" size="sm" />}
                {item.label}
              </button>
            ) : (
              // Non-clickable item
              <span className="text-gray-500 dark:text-gray-400 flex items-center">
                {item.icon && <Icon name={item.icon} className="text-gray-500 dark:text-gray-400 mr-1" size="sm" />}
                {item.label}
              </span>
            )}
          </div>

          {/* Separator */}
          {index < items.length - 1 && (
            <span className="text-gray-400" aria-hidden="true">
              {separator}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
  );
}
