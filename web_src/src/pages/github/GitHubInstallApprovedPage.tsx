import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import {
  GITHUB_INSTALL_APPROVED_ACTION,
  GITHUB_INSTALL_APPROVED_BODY,
  GITHUB_INSTALL_APPROVED_TITLE,
} from "@/lib/githubInstallRequestCopy";
import { useFactoriesThemeClass } from "@/pages/factories/lib/useFactoriesThemeClass";

export function GitHubInstallApprovedPage() {
  useFactoriesThemeClass();
  usePageTitle([GITHUB_INSTALL_APPROVED_TITLE]);
  useReportPageReady(true);

  return (
    <div className="fixed inset-0 bg-background text-foreground" data-testid="github-install-approved">
      <div className="flex h-full items-center justify-center px-6">
        <div className="w-full max-w-md space-y-6 text-center">
          <header className="space-y-3">
            <h1 className="workspace-page-title font-semibold">{GITHUB_INSTALL_APPROVED_TITLE}</h1>
            <p className="text-[15px] leading-6 text-muted-foreground">{GITHUB_INSTALL_APPROVED_BODY}</p>
          </header>
          <Button asChild>
            <Link to="/" data-testid="github-install-approved-open">
              {GITHUB_INSTALL_APPROVED_ACTION}
            </Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
