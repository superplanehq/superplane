import { Check, FlaskConical, KeyRound, Package, Ruler, ShieldCheck, Trash2, type LucideIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import type { QualityDomain, QualityTemplate } from "@/pages/factories/verification/types";
import { QUALITY_DOMAIN_LABELS } from "@/pages/factories/verification/types";

const DOMAIN_ICONS: Record<QualityDomain, LucideIcon> = {
  "type-safety": ShieldCheck,
  tests: FlaskConical,
  secrets: KeyRound,
  "dead-code": Trash2,
  "file-size": Ruler,
  dependencies: Package,
};

interface QualityTemplateGalleryProps {
  templates: QualityTemplate[];
  onInstall: (templateId: string) => void;
}

/**
 * Gallery of the installable quality templates. Each template is one canvas
 * that runs standalone or as a check inside a verification suite.
 */
export function QualityTemplateGallery({ templates, onInstall }: QualityTemplateGalleryProps) {
  return (
    <section className="flex flex-col gap-3" aria-label="Quality templates">
      <div className="flex flex-col gap-0.5">
        <h3 className="workspace-section-title text-foreground">Quality templates</h3>
        <p className="text-[12px] text-muted-foreground">
          Install a template to add its checks to your factory. Each template also runs standalone.
        </p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {templates.map((template) => (
          <QualityTemplateCard key={template.id} template={template} onInstall={onInstall} />
        ))}
      </div>
    </section>
  );
}

function QualityTemplateCard({
  template,
  onInstall,
}: {
  template: QualityTemplate;
  onInstall: (templateId: string) => void;
}) {
  const DomainIcon = DOMAIN_ICONS[template.domain];
  return (
    <article className={cn(factoryCardClassName, "flex flex-col gap-3 p-4")} aria-label={template.name}>
      <div className="flex items-center gap-2.5">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground">
          <DomainIcon className="size-4.5" aria-hidden />
        </span>
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-[13px] font-medium text-foreground">{template.name}</span>
          <span className="text-[12px] text-muted-foreground">{QUALITY_DOMAIN_LABELS[template.domain]}</span>
        </div>
      </div>
      <p className="workspace-body-text flex-1 text-muted-foreground">{template.description}</p>
      <div className="flex items-center justify-between gap-2 border-t border-border pt-3">
        <span className="text-[12px] text-muted-foreground">{template.checksPerRun} checks per run</span>
        {template.installed ? (
          <span className="inline-flex items-center gap-1 text-[13px] font-medium text-emerald-700 dark:text-emerald-300">
            <Check className="size-4" aria-hidden />
            Installed
          </span>
        ) : (
          <Button size="sm" variant="outline" onClick={() => onInstall(template.id)}>
            Install template
          </Button>
        )}
      </div>
    </article>
  );
}
