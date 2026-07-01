import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { cn } from "@/lib/utils";
import { MoveLeft } from "lucide-react";
import { Link } from "react-router-dom";

interface SetupHeaderProps {
  integrationsHref: string;
  integrationName: string;
  iconSlug?: string;
  setupPageTitle: string;
  hasCreatedIntegration: boolean;
}

export function SetupHeader({
  integrationsHref,
  integrationName,
  iconSlug,
  setupPageTitle,
  hasCreatedIntegration,
}: SetupHeaderProps) {
  return (
    <header className={cn("space-y-3", !hasCreatedIntegration && "mb-6 px-4 sm:px-6")}>
      <nav className="text-xs text-gray-500 dark:text-gray-400" aria-label="Setup navigation">
        <Link
          to={integrationsHref}
          className="inline-flex items-center gap-1.5 font-medium leading-none text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-100"
        >
          <MoveLeft aria-hidden className="size-[1em] shrink-0 opacity-80" />
          Integrations
        </Link>
      </nav>

      <div className="flex w-full min-w-0 items-center gap-3">
        <IntegrationIcon
          integrationName={integrationName}
          iconSlug={iconSlug}
          className="h-6 w-6 shrink-0 text-gray-700 dark:text-gray-300"
        />
        <h4 className="min-w-0 truncate text-2xl font-medium text-gray-900 dark:text-gray-100">{setupPageTitle}</h4>
      </div>
    </header>
  );
}
