import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";

const copy = FIRST_RUN_COPY.bitbucket;

export function BitbucketCopyInstallLink({ installUrl, onCopy }: { installUrl: string; onCopy?: () => Promise<void> }) {
  const [copied, setCopied] = useState(false);
  const [copyFailed, setCopyFailed] = useState(false);
  const copyLink = async () => {
    if (!onCopy) return;
    setCopyFailed(false);
    try {
      await onCopy();
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
      setCopyFailed(true);
    }
  };
  return (
    <div className="space-y-2">
      <p className="text-[12px] text-muted-foreground">
        {copy.wrongAccountNote}{" "}
        <Button
          type="button"
          variant="link"
          className="h-auto p-0 align-baseline text-[12px] font-medium text-inherit underline underline-offset-2 hover:no-underline"
          disabled={!installUrl}
          onClick={() => void copyLink()}
          data-testid="first-run-bitbucket-copy-install-link"
        >
          {copied ? copy.copiedInstallLink : copy.copyInstallLink}
        </Button>
      </p>
      {copyFailed ? (
        <div className="space-y-1" role="alert" data-testid="first-run-bitbucket-copy-error">
          <p className="text-[13px] text-destructive">{copy.copyInstallLinkError}</p>
          <p className="text-[12px] text-muted-foreground">{copy.copyInstallLinkRecovery}</p>
          {installUrl ? (
            <div className="space-y-1">
              <Label htmlFor="first-run-bitbucket-install-link" className="text-[12px] font-medium">
                {copy.installLinkLabel}
              </Label>
              <Input
                id="first-run-bitbucket-install-link"
                readOnly
                value={installUrl}
                onFocus={(event) => event.currentTarget.select()}
                data-testid="first-run-bitbucket-install-link"
                className="h-8 font-mono text-[12px]"
              />
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
