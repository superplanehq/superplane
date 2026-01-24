import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Undo2, Palette, Home, ChevronDown, Copy, Download } from "lucide-react";
import { Button } from "../button";
import { Switch } from "../switch";
import { useWorkflows } from "@/hooks/useWorkflowData";
import { useParams, useNavigate } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/ui/select";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  href?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
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
  onExportYamlCopy?: () => void;
  onExportYamlDownload?: () => void;
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
  onExportYamlCopy,
  onExportYamlDownload,
}: HeaderProps) {
  const { workflowId } = useParams<{ workflowId?: string }>();
  const navigate = useNavigate();
  const { data: workflows = [], isLoading: workflowsLoading } = useWorkflows(organizationId || "");
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [exportAction, setExportAction] = useState<string>("");
  const menuRef = useRef<HTMLDivElement | null>(null);

  // Get the workflow name from the workflows list if workflowId is available
  // Otherwise, use breadcrumbs[1] which is always the workflow name (index 0 is "Canvases")
  // Fall back to the last breadcrumb item if neither is available
  const currentWorkflowName = (() => {
    if (workflowId) {
      const workflow = workflows.find((w) => w.metadata?.id === workflowId);
      if (workflow?.metadata?.name) {
        return workflow.metadata.name;
      }
    }
    // breadcrumbs[1] is always the workflow name (index 0 is "Canvases")
    if (breadcrumbs.length > 1 && breadcrumbs[1]?.label) {
      return breadcrumbs[1].label;
    }
    // Fall back to last breadcrumb if no workflow name found
    return breadcrumbs.length > 0 ? breadcrumbs[breadcrumbs.length - 1].label : "";
  })();

  useEffect(() => {
    if (!isMenuOpen) return;

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMenuOpen(false);
      }
    };

    const listenerOptions: AddEventListenerOptions = { capture: true };

    document.addEventListener("mousedown", handlePointerDown, listenerOptions);
    document.addEventListener("touchstart", handlePointerDown, listenerOptions);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown, listenerOptions);
      document.removeEventListener("touchstart", handlePointerDown, listenerOptions);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isMenuOpen]);

  const handleWorkflowClick = (selectedWorkflowId: string) => {
    if (selectedWorkflowId && organizationId) {
      setIsMenuOpen(false);
      navigate(`/${organizationId}/workflows/${selectedWorkflowId}`);
    }
  };

  return (
    <>
      <header className="bg-white dark:bg-neutral-900 border-b border-slate-950/15 dark:border-neutral-700">
        <div className="relative flex items-center justify-between h-12 px-4">
          <div className="flex items-center gap-3">
            <OrganizationMenuButton organizationId={organizationId} onLogoClick={onLogoClick} />

            {/* Canvas Dropdown */}
            {organizationId && (
              <div className="relative flex items-center" ref={menuRef}>
                <button
                  onClick={() => setIsMenuOpen((prev) => !prev)}
                  className="flex items-center gap-1 cursor-pointer h-7"
                  aria-label="Open canvas menu"
                  aria-expanded={isMenuOpen}
                  disabled={workflowsLoading}
                >
                  <span className="text-sm text-gray-800 dark:text-gray-200 font-medium">
                    {currentWorkflowName || (workflowsLoading ? "Loading..." : "Select canvas")}
                  </span>
                  <ChevronDown
                    size={16}
                    className={`text-gray-400 dark:text-gray-500 transition-transform ${isMenuOpen ? "rotate-180" : ""}`}
                  />
                </button>
                {isMenuOpen && !workflowsLoading && (
                  <div className="absolute left-0 top-13 z-50 min-w-[15rem] w-max rounded-md outline outline-slate-950/20 dark:outline-neutral-700 bg-white dark:bg-neutral-800 shadow-lg">
                    <div className="px-4 pt-3 pb-4">
                      {/* All Canvases Link */}
                      <div className="mb-2">
                        <a
                          href={organizationId ? `/${organizationId}` : "/"}
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
                          onClick={() => setIsMenuOpen(false)}
                        >
                          <Home size={16} className="text-gray-500 dark:text-gray-400 transition group-hover:text-gray-800 dark:group-hover:text-gray-200" />
                          <span>All Canvases</span>
                        </a>
                      </div>
                      {/* Divider */}
                      <div className="border-b border-gray-300 dark:border-neutral-600 mb-2"></div>
                      {/* Canvas List */}
                      <div className="mt-2 flex flex-col">
                        {workflows.map((workflow) => {
                          const isSelected = workflow.metadata?.id === workflowId;
                          return (
                            <button
                              key={workflow.metadata?.id}
                              type="button"
                              onClick={() => handleWorkflowClick(workflow.metadata?.id || "")}
                              className={`group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-left ${
                                isSelected
                                  ? "bg-sky-100 dark:bg-sky-900/30 text-gray-800 dark:text-gray-200"
                                  : "text-gray-500 dark:text-gray-400 hover:bg-sky-100 dark:hover:bg-sky-900/30 hover:text-gray-800 dark:hover:text-gray-200"
                              }`}
                            >
                              {isSelected ? (
                                <Palette size={16} className="text-gray-800 dark:text-gray-200 transition group-hover:text-gray-800 dark:group-hover:text-gray-200" />
                              ) : (
                                <span className="w-4" />
                              )}
                              <span>{workflow.metadata?.name || "Unnamed Canvas"}</span>
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Right side - Auto-save toggle and Save button */}
          <div className="flex items-center gap-3">
            {onExportYamlCopy && onExportYamlDownload && (
              <Select
                value={exportAction || undefined}
                onValueChange={(value) => {
                  setExportAction(value);
                  if (value === "copy") {
                    onExportYamlCopy();
                  }
                  if (value === "download") {
                    onExportYamlDownload();
                  }
                  setExportAction("");
                }}
              >
                <SelectTrigger className="w-20">
                  <SelectValue placeholder="YAML" />
                </SelectTrigger>
                <SelectContent align="end">
                  <SelectItem value="copy">
                    <span className="flex items-center gap-2">
                      <Copy className="h-3.5 w-3.5" />
                      Copy to Clipboard
                    </span>
                  </SelectItem>
                  <SelectItem value="download">
                    <span className="flex items-center gap-2">
                      <Download className="h-3.5 w-3.5" />
                      Download File
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
            )}
            {unsavedMessage && (
              <span className="text-xs font-medium text-yellow-700 dark:text-yellow-400 bg-orange-100 dark:bg-orange-900/30 px-2 py-1 rounded hidden sm:inline">
                {unsavedMessage}
              </span>
            )}
            {onToggleAutoSave && (
              <div className="flex items-center gap-2">
                <label htmlFor="auto-save-toggle" className="text-sm text-gray-800 dark:text-gray-200 hidden sm:inline">
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
