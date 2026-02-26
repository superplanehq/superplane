import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Undo2, Palette, Home, ChevronDown, Copy, Download } from "lucide-react";
import { Button } from "../button";
import { Switch } from "../switch";
import { useCanvases } from "@/hooks/useCanvasData";
import { useParams, useNavigate } from "react-router-dom";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

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
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  isAutoSaveEnabled?: boolean;
  onToggleAutoSave?: () => void;
  autoSaveDisabled?: boolean;
  autoSaveDisabledTooltip?: string;
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
  saveDisabled,
  saveDisabledTooltip,
  isAutoSaveEnabled,
  onToggleAutoSave,
  autoSaveDisabled,
  autoSaveDisabledTooltip,
  onExportYamlCopy,
  onExportYamlDownload,
}: HeaderProps) {
  const { workflowId } = useParams<{ workflowId?: string }>();
  const navigate = useNavigate();
  const { data: workflows = [], isLoading: workflowsLoading } = useCanvases(organizationId || "");
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
      navigate(`/${organizationId}/canvases/${selectedWorkflowId}`);
    }
  };

  const wrapWithTooltip = (disabled: boolean | undefined, message: string | undefined, child: ReactNode) => {
    if (!disabled || !message) return child;
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{child}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{message}</TooltipContent>
      </Tooltip>
    );
  };

  return (
    <>
      <header className="bg-white border-b border-slate-950/15">
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
                  <span className="text-sm text-gray-800 font-medium">
                    {currentWorkflowName || (workflowsLoading ? "Loading..." : "Select canvas")}
                  </span>
                  <ChevronDown
                    size={16}
                    className={`text-gray-400 transition-transform ${isMenuOpen ? "rotate-180" : ""}`}
                  />
                </button>
                {isMenuOpen && !workflowsLoading && (
                  <div className="absolute left-0 top-13 z-50 min-w-[15rem] w-max rounded-md outline outline-slate-950/20 bg-white shadow-lg">
                    <div className="px-4 pt-3 pb-4">
                      {/* All Canvases Link */}
                      <div className="mb-2">
                        <a
                          href={organizationId ? `/${organizationId}` : "/"}
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:text-gray-800"
                          onClick={() => setIsMenuOpen(false)}
                        >
                          <Home size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                          <span>All Canvases</span>
                        </a>
                      </div>
                      {/* Divider */}
                      <div className="border-b border-gray-300 mb-2"></div>
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
                                  ? "bg-sky-100 text-gray-800"
                                  : "text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                              }`}
                            >
                              {isSelected ? (
                                <Palette size={16} className="text-gray-800 transition group-hover:text-gray-800" />
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
                <SelectTrigger className="h-5 w-fit min-w-0 rounded-md border-gray-300 px-1 py-0 text-xs font-mono text-gray-500 data-[placeholder]:text-gray-500 shadow-none [&>svg]:hidden">
                  <SelectValue placeholder=".yaml" />
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
              <span className="text-xs font-medium text-yellow-700 bg-orange-100 px-2 py-1 rounded hidden sm:inline">
                {unsavedMessage}
              </span>
            )}
            {onToggleAutoSave &&
              wrapWithTooltip(
                autoSaveDisabled,
                autoSaveDisabledTooltip,
                <div className="flex items-center gap-2">
                  <label
                    htmlFor="auto-save-toggle"
                    className={`text-sm hidden sm:inline ${autoSaveDisabled ? "text-gray-400" : "text-gray-800"}`}
                  >
                    Auto-save
                  </label>
                  <Switch
                    id="auto-save-toggle"
                    checked={isAutoSaveEnabled}
                    onCheckedChange={onToggleAutoSave}
                    disabled={autoSaveDisabled}
                  />
                </div>,
              )}
            {onUndo && canUndo && (
              <Button onClick={onUndo} size="sm" variant="outline">
                <Undo2 />
                Revert
              </Button>
            )}
            {onSave &&
              !saveButtonHidden &&
              wrapWithTooltip(
                saveDisabled,
                saveDisabledTooltip,
                <Button
                  onClick={onSave}
                  size="sm"
                  variant={saveIsPrimary ? "default" : "outline"}
                  data-testid="save-canvas-button"
                  disabled={saveDisabled}
                >
                  Save
                </Button>,
              )}
          </div>
        </div>
      </header>
    </>
  );
}
