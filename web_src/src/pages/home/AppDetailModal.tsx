import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { AppLogo } from "@/ui/componentSidebar/AppLogo";
import { resolveLucideIcon } from "@/lib/iconRegistry";
import { ArrowLeft, ArrowRight, ExternalLink, Plus } from "lucide-react";
import {
  homeModalFooterEdgeClassName,
  homeModalHeaderEdgeClassName,
  homeModalOverlayClassName,
  homeModalPanelClassName,
  homeModalTitleClassName,
  homeTagLargeClassName,
} from "./homePageStyles";

export interface AppEntry {
  repo: string;
  icon: string;
  title: string;
  description: string;
  integrations: string[];
  tags: string[];
  requirements: string[];
  agentInstructions: string;
  agentInitialMessage?: string;
}

interface AppDetailModalProps {
  app: AppEntry;
  busy: boolean;
  onBack: () => void;
  onInstall: (e: React.MouseEvent) => void;
  onClose: () => void;
}

export function AppDetailModal({ app, busy, onBack, onInstall, onClose }: AppDetailModalProps) {
  const repoUrl = `https://${app.repo}`;

  return (
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[12vh] sm:pt-[14vh]">
      <div className={homeModalOverlayClassName} onClick={onClose} />
      <div className={homeModalPanelClassName}>
        <div className={homeModalHeaderEdgeClassName}>
          <button
            type="button"
            onClick={onBack}
            className="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Back
          </button>
        </div>

        <div className="px-6 py-5">
          <div className="flex items-start gap-4">
            <div className="shrink-0">
              <LeadIcon icon={app.icon} integrations={app.integrations} size="lg" />
            </div>
            <div className="min-w-0 flex-1">
              <h3 className={homeModalTitleClassName}>{app.title}</h3>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">
                <IntegrationIcons integrations={app.integrations} />
                {app.tags.map((tag) => (
                  <span key={tag} className={homeTagLargeClassName}>
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          </div>

          <div className="mt-5">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
              Description
            </h4>
            <p className="mt-1.5 text-sm leading-relaxed text-gray-800 dark:text-gray-300">{app.description}</p>
          </div>

          {app.requirements.length > 0 && (
            <div className="mt-4">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                Requirements
              </h4>
              <ul className="mt-1.5 space-y-1">
                {app.requirements.map((req) => (
                  <li key={req} className="flex items-start gap-2 text-sm text-slate-600 dark:text-gray-400">
                    <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-slate-400 dark:bg-gray-500" />
                    {req}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        <div className={homeModalFooterEdgeClassName}>
          <a
            href={repoUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            View on GitHub
          </a>
          <Button onClick={onInstall} disabled={busy}>
            Install
            <ArrowRight />
          </Button>
        </div>
      </div>
    </div>
  );
}

export function LeadIcon({
  icon,
  integrations,
  size = "sm",
}: {
  icon?: string;
  integrations: string[];
  size?: "sm" | "lg";
}) {
  const iconName = icon || integrations[0];
  const cls = size === "lg" ? "h-8 w-8" : "h-5 w-5";
  if (!iconName) return <Plus className={`${cls} text-slate-400 dark:text-gray-500`} />;
  const iconSrc = getIntegrationIconSrc(iconName.toLowerCase());
  if (iconSrc) return <AppLogo src={iconSrc} alt={iconName} className={cls} />;
  const FallbackIcon = resolveLucideIcon(iconName);
  return <FallbackIcon className={`${cls} text-slate-500 dark:text-gray-400`} />;
}

export function IntegrationIcons({ integrations }: { integrations: string[] }) {
  if (integrations.length === 0) return null;
  return (
    <div className="flex items-center gap-1 shrink-0">
      {integrations.map((name) => {
        const iconSrc = getIntegrationIconSrc(name.toLowerCase());
        if (!iconSrc) return null;
        return (
          <Tooltip key={name}>
            <TooltipTrigger asChild>
              <span className="inline-block h-3.5 w-3.5 shrink-0">
                <AppLogo src={iconSrc} alt={name} className="h-full w-full object-contain" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom" className="z-[300]">
              <span className="capitalize">{name}</span>
            </TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}
