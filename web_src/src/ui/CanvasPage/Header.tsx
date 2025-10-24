import { ChevronRight } from "lucide-react";
import SuperplaneLogo from "@/assets/superplane.svg";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
}

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
}

export function Header({ breadcrumbs }: HeaderProps) {
  return (
    <header className="absolute top-0 left-0 right-0 z-20 bg-white border-b border-gray-200 shadow-sm">
      <div className="flex items-center justify-between h-12 px-6">
        {/* Logo */}
        <div className="flex items-center">
          <img
            src={SuperplaneLogo}
            alt="Logo"
            className="w-8 h-8"
          />
        </div>

        {/* Breadcrumbs */}
        <div className="flex items-center space-x-2 text-[15px] text-gray-500">
          {breadcrumbs.map((item, index) => (
            <div key={index} className="flex items-center">
              {index > 0 && (
                <div className="w-2 mx-2 font-bold">/</div>
              )}
              {item.onClick ? (
                <button
                  onClick={item.onClick}
                  className="hover:text-black transition-colors"
                >
                  {item.label}
                </button>
              ) : (
                <span className={`font-medium ${index === breadcrumbs.length - 1 ? "text-black" : ""}`}>{item.label}</span>
              )}
            </div>
          ))}
        </div>

        {/* Right side - placeholder for future actions */}
        <div className="w-8"></div>
      </div>
    </header>
  );
}