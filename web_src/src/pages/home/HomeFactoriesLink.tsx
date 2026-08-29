import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { ArrowRight, Factory as FactoryIcon } from "lucide-react";
import { factoryListPath } from "@/pages/factories/lib/factoryPagePaths";
import { homeListCardClassName } from "./homePageStyles";

interface HomeFactoriesLinkProps {
  organizationId: string;
}

export function HomeFactoriesLink({ organizationId }: HomeFactoriesLinkProps) {
  return (
    <section className="mb-10" data-testid="home-factories-link">
      <Link
        href={factoryListPath(organizationId)}
        className={cn(
          homeListCardClassName,
          "flex items-center gap-3 px-4 py-3 no-underline hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400",
        )}
        data-testid="home-factories-link-anchor"
      >
        <span
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300"
          aria-hidden
        >
          <FactoryIcon className="h-5 w-5" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">Navigate to Workspaces</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">Manage tasks, automations, and apps.</p>
        </div>
        <ArrowRight className="h-4 w-4 shrink-0 text-gray-400" aria-hidden />
      </Link>
    </section>
  );
}
