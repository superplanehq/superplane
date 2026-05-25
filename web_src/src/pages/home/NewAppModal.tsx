import { Dialog, DialogBody, DialogTitle } from "@/components/Dialog/dialog";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { INTEGRATION_APP_LOGO_MAP } from "@/ui/componentSidebar/integrationIconMaps";
import { Plus } from "lucide-react";
import templateManifest from "../../../../templates/manifest.json";
import { useCreateApp } from "./useCreateApp";
import { useInstallTemplate } from "./useInstallTemplate";

interface TemplateEntry {
  repo: string;
  title: string;
  description: string;
  integrations: string[];
  tags: string[];
}

const templates: TemplateEntry[] = templateManifest;

interface NewAppModalProps {
  open: boolean;
  onClose: () => void;
}

export function NewAppModal({ open, onClose }: NewAppModalProps) {
  const { createApp, isSaving } = useCreateApp({ onCreated: onClose });
  const { installTemplate, isInstalling } = useInstallTemplate();

  const handleBlankCreate = () => {
    if (isSaving) return;
    void createApp(generateCanvasName());
  };

  const handleTemplateClick = (repo: string) => {
    if (isInstalling) return;
    void installTemplate(repo);
  };

  const busy = isSaving || isInstalling;

  return (
    <Dialog open={open} onClose={onClose} size="md">
      <DialogTitle>Create New App</DialogTitle>
      <DialogBody>
        <button
          type="button"
          disabled={busy}
          onClick={handleBlankCreate}
          className="flex w-full items-center gap-3 rounded-lg border border-slate-200 p-4 text-left transition-colors hover:bg-slate-50 disabled:opacity-50"
        >
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-100">
            <Plus className="h-5 w-5 text-slate-600" />
          </div>
          <div>
            <p className="text-sm font-medium text-slate-900">Start from scratch</p>
            <p className="text-xs text-slate-500">Create a blank canvas</p>
          </div>
        </button>

        {templates.length > 0 && (
          <>
            <div className="relative my-4">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-slate-200" />
              </div>
              <div className="relative flex justify-center text-xs">
                <span className="bg-white px-2 text-slate-500">Or start from a template</span>
              </div>
            </div>

            <div className="flex flex-col gap-2">
              {templates.map((template) => (
                <button
                  key={template.repo}
                  type="button"
                  disabled={busy}
                  onClick={() => handleTemplateClick(template.repo)}
                  className="flex w-full items-start gap-3 rounded-lg border border-slate-200 p-4 text-left transition-colors hover:bg-slate-50 disabled:opacity-50"
                >
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-100">
                    <IntegrationStack integrations={template.integrations} />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-slate-900">{template.title}</p>
                    <p className="mt-0.5 text-xs text-slate-500 line-clamp-2">{template.description}</p>
                    <div className="mt-2 flex flex-wrap gap-1">
                      {template.integrations.map((integration) => (
                        <IntegrationBadge key={integration} name={integration} />
                      ))}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </>
        )}
      </DialogBody>
    </Dialog>
  );
}

function IntegrationStack({ integrations }: { integrations: string[] }) {
  const first = integrations[0];
  if (!first) return <Plus className="h-5 w-5 text-slate-400" />;

  const icon = INTEGRATION_APP_LOGO_MAP[first.toLowerCase()];
  if (!icon) return <Plus className="h-5 w-5 text-slate-400" />;

  return <img src={icon} alt={first} className="h-6 w-6" />;
}

function IntegrationBadge({ name }: { name: string }) {
  const icon = INTEGRATION_APP_LOGO_MAP[name.toLowerCase()];
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-600">
      {icon && <img src={icon} alt={name} className="h-3 w-3" />}
      {name}
    </span>
  );
}
