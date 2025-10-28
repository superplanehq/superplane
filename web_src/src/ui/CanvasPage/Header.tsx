import SuperplaneLogo from "@/assets/superplane.svg";
import { resolveIcon } from "@/lib/utils";
import { Button } from "../button";
import { Save } from "lucide-react";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
}

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
  onSave?: () => void;
}

export function Header({ breadcrumbs, onSave }: HeaderProps) {
  return (
    <header className="absolute top-0 left-0 right-0 z-20 bg-white border-b border-gray-200">
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
          {breadcrumbs.map((item, index) => {
            const IconComponent = item.iconSlug ? resolveIcon(item.iconSlug) : null;

            return (
              <div key={index} className="flex items-center">
                {index > 0 && (
                  <div className="w-2 mx-2">/</div>
                )}
                {item.onClick ? (
                  <button
                    onClick={item.onClick}
                    className="hover:text-black transition-colors flex items-center gap-2"
                  >
                    {item.iconSrc && (
                      <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                        <img src={item.iconSrc} alt="" className="w-5 h-5" />
                      </div>
                    )}
                    {IconComponent && (
                      <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                        <IconComponent size={16} className={item.iconColor || ""} />
                      </div>
                    )}
                    {item.label}
                  </button>
                ) : (
                  <span className={`flex items-center gap-2 ${index === breadcrumbs.length - 1 ? "text-black font-medium" : ""}`}>
                    {item.iconSrc && (
                      <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                        <img src={item.iconSrc} alt="" className="w-5 h-5" />
                      </div>
                    )}
                    {IconComponent && (
                      <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                        <IconComponent size={16} className={item.iconColor || ""} />
                      </div>
                    )}
                    {item.label}
                  </span>
                )}
              </div>
            );
          })}
        </div>

        {/* Right side - Save button */}
        {onSave ? (
          <Button onClick={onSave} size="sm" variant="outline">
            <Save />
            Save
          </Button>
        ) : (
          <div className="w-8"></div>
        )}
      </div>
    </header>
  );
}