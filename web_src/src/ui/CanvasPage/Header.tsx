import SuperplaneLogo from "@/assets/superplane.svg";
import { resolveIcon } from "@/lib/utils";
import { Save, Trash2 } from "lucide-react";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "../button";

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
  onDelete?: () => void;
  onLogoClick?: () => void;
}

export function Header({ breadcrumbs, onSave, onDelete, onLogoClick }: HeaderProps) {
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const handleDeleteClick = () => {
    setShowDeleteModal(true);
  };

  const handleConfirmDelete = () => {
    setShowDeleteModal(false);
    onDelete?.();
  };

  return (
    <>
      <header className="bg-white border-b border-gray-200">
        <div className="flex items-center justify-between h-12 px-6">
          {/* Logo */}
          <div className="flex items-center">
            {onLogoClick ? (
              <button
                onClick={onLogoClick}
                className="cursor-pointer hover:opacity-80 transition-opacity"
                aria-label="Go to organization homepage"
              >
                <img
                  src={SuperplaneLogo}
                  alt="Logo"
                  className="w-8 h-8"
                />
              </button>
            ) : (
              <img
                src={SuperplaneLogo}
                alt="Logo"
                className="w-8 h-8"
              />
            )}
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

          {/* Right side - Save and Delete buttons */}
          <div className="flex items-center gap-4 text-[15px] text-gray-500">
            {onSave && (
              <button
                onClick={onSave}
                className="hover:text-black transition-colors flex items-center gap-1.5"
              >
                <Save size={16} />
                Save
              </button>
            )}
            {onDelete && (
              <button
                onClick={handleDeleteClick}
                className="hover:text-red-600 transition-colors flex items-center gap-1.5"
              >
                <Trash2 size={16} />
                Delete
              </button>
            )}
            {!onSave && !onDelete && <div className="w-8"></div>}
          </div>
        </div>
      </header>

      {/* Delete Confirmation Modal */}
      <Dialog open={showDeleteModal} onOpenChange={setShowDeleteModal}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Workflow</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this workflow? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteModal(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}