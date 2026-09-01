import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import { Avatar } from "@/components/Avatar/avatar";
import { Button } from "@/components/ui/button";
import type { ConnectedAccountProvider } from "@/contexts/accountContextState";
import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";

interface GitHubAccountConnectionProps {
  providers: ConnectedAccountProvider[];
  impersonating?: boolean;
}

const linkResultMessages = {
  success: { type: "success", message: "GitHub profile connected." },
  conflict: {
    type: "error",
    message: "This account or GitHub profile already has a different connection.",
  },
  denied: { type: "info", message: "GitHub connection canceled." },
  failure: {
    type: "error",
    message: "SuperPlane could not connect to GitHub. Try again.",
  },
} as const;

export function GitHubAccountConnection({ providers, impersonating = false }: GitHubAccountConnectionProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const github = providers.find((provider) => provider.provider === "github");

  useEffect(() => {
    const search = new URLSearchParams(location.search);
    const provider = search.get("provider");
    const result = search.get("provider_link") as keyof typeof linkResultMessages | null;
    if (provider !== "github" || !result || !linkResultMessages[result]) {
      return;
    }

    const notification = linkResultMessages[result];
    if (notification.type === "success") {
      showSuccessToast(notification.message);
    } else if (notification.type === "info") {
      showInfoToast(notification.message);
    } else {
      showErrorToast(notification.message);
    }

    search.delete("provider");
    search.delete("provider_link");
    navigate(
      {
        pathname: location.pathname,
        search: search.size > 0 ? `?${search.toString()}` : "",
      },
      { replace: true },
    );
  }, [location.pathname, location.search, navigate]);

  const returnPath = location.pathname;
  const connectPath = `/account/providers/github/connect?redirect=${encodeURIComponent(returnPath)}`;

  return (
    <div className="flex items-center justify-between gap-4 rounded-md border border-border p-4">
      <div className="flex min-w-0 items-center gap-3">
        <Avatar
          src={github?.avatar_url || undefined}
          initials={github?.avatar_url ? undefined : "GH"}
          alt={github?.display_name || github?.username || "GitHub"}
          className="size-10"
        />
        <div className="min-w-0">
          <p className="text-[13px] font-medium text-foreground">GitHub</p>
          <p className="truncate text-[12px] text-muted-foreground">
            {github ? `@${github.username}` : "Connect your profile to match repository activity to your account."}
          </p>
        </div>
      </div>

      {github ? (
        <span className="shrink-0 text-[12px] font-medium text-emerald-600 dark:text-emerald-400">Connected</span>
      ) : (
        <Button asChild={!impersonating} variant="outline" size="sm" disabled={impersonating}>
          {impersonating ? (
            <span>Connect GitHub</span>
          ) : (
            <a href={connectPath} data-testid="connect-github-profile">
              Connect GitHub
            </a>
          )}
        </Button>
      )}
    </div>
  );
}
