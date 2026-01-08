import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { resolveIcon } from "@/lib/utils";
import { Undo2 } from "lucide-react";
import { Button } from "../button";
import { Switch } from "../switch";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  href?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
}

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
  onSave?: () => void;
  onUndo?: () => void;
  canUndo?: boolean;
  onLogoClick?: () => void;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  isAutoSaveEnabled?: boolean;
  onToggleAutoSave?: () => void;
}

export function Header({
  breadcrumbs,
  onSave,
  onUndo,
  canUndo,
  onLogoClick,
  organizationId,
  unsavedMessage,
  saveIsPrimary,
  saveButtonHidden,
  isAutoSaveEnabled,
  onToggleAutoSave,
}: HeaderProps) {
  return (
    <>
      <header className="bg-white border-b border-border">
        <div className="relative flex items-center justify-between h-12 px-4">
          <OrganizationMenuButton organizationId={organizationId} onLogoClick={onLogoClick} />

          {/* Breadcrumbs - Absolutely centered */}
          <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 flex items-center space-x-1 text-sm text-gray-500">
            {breadcrumbs.map((item, index) => {
              const IconComponent = item.iconSlug ? resolveIcon(item.iconSlug) : null;

              return (
                <div key={index} className="flex items-center">
                  {index > 0 && <div className="w-2 mx-1">/</div>}
                  {item.href || item.onClick ? (
                    <a
                      href={item.href}
                      onClick={item.onClick}
                      className="hover:text-gray-800 transition-colors flex items-center gap-2"
                    >
                      {item.iconSrc && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
                          <img src={item.iconSrc} alt="" className="w-5 h-5" />
                        </div>
                      )}
                      {IconComponent && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
                          <IconComponent size={16} className={item.iconColor || ""} />
                        </div>
                      )}
                      {item.label}
                    </a>
                  ) : (
                    <span
                      className={`flex items-center gap-1 ${
                        index === breadcrumbs.length - 1 ? "text-gray-800 font-medium" : ""
                      }`}
                    >
                      {item.iconSrc && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
                          <img src={item.iconSrc} alt="" className="w-5 h-5" />
                        </div>
                      )}
                      {IconComponent && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
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

          {/* Right side - Auto-save toggle and Save button */}
          <div className="flex items-center gap-3">
            {unsavedMessage && (
              <span className="text-xs font-medium text-yellow-700 bg-orange-100 px-2 py-1 rounded hidden sm:inline">
                {unsavedMessage}
              </span>
            )}
            {onToggleAutoSave && (
              <div className="flex items-center gap-2">
                <label htmlFor="auto-save-toggle" className="text-xs text-gray-600 hidden sm:inline">
                  Auto-save
                </label>
                <Switch id="auto-save-toggle" checked={isAutoSaveEnabled} onCheckedChange={onToggleAutoSave} />
              </div>
            )}
            {onUndo && canUndo && (
              <Button onClick={onUndo} size="sm" variant="outline">
                <Undo2 />
                Revert
              </Button>
            )}
            {onSave && !saveButtonHidden && (
              <Button
                onClick={onSave}
                size="sm"
                variant={saveIsPrimary ? "default" : "outline"}
                data-testid="save-canvas-button"
              >
                Save
              </Button>
            )}
          </div>
        </div>
      </header>
    </>
  );
}
