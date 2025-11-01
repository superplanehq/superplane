import SuperplaneLogo from "@/assets/superplane.svg";
import { resolveIcon } from "@/lib/utils";
import { Save, Trash2, AlertTriangle } from "lucide-react";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
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
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  // Get the workflow name from the last breadcrumb
  const workflowName = breadcrumbs[breadcrumbs.length - 1]?.label || "";

  const handleDeleteClick = () => {
    setShowDeleteModal(true);
    setDeleteConfirmation("");
  };

  const handleConfirmDelete = () => {
    setShowDeleteModal(false);
    setDeleteConfirmation("");
    onDelete?.();
  };

  const isDeleteEnabled = deleteConfirmation === workflowName;

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
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <div className="flex items-center gap-3 mb-2">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
                <AlertTriangle className="h-6 w-6 text-red-600" />
              </div>
              <DialogTitle className="text-xl">Delete Workflow</DialogTitle>
            </div>
            <DialogDescription className="text-base space-y-3 pt-2">
              <p className="text-gray-700 font-medium">
                This action cannot be undone. This will permanently delete the workflow and all associated events and executions.
              </p>
              <p className="text-gray-600">
                Please type <span className="font-mono font-semibold text-gray-900 bg-gray-100 px-1.5 py-0.5 rounded">{workflowName}</span> to confirm.
              </p>
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            <Input
              type="text"
              placeholder={`Enter "${workflowName}" to confirm`}
              value={deleteConfirmation}
              onChange={(e) => setDeleteConfirmation(e.target.value)}
              className="font-mono"
              autoFocus
            />
          </div>

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
              disabled={!isDeleteEnabled}
              className="bg-red-600 hover:bg-red-700 text-white disabled:bg-gray-300 disabled:text-gray-500 disabled:cursor-not-allowed"
            >
              <Trash2 className="h-4 w-4 mr-1.5" />
              Delete Workflow
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}