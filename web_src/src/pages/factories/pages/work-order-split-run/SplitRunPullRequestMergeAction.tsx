import { useEffect, useState } from "react";
import { ChevronDown } from "lucide-react";

import type {
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestMergeability,
  FactoryPullRequestMergeabilityMergeMethod,
} from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { ButtonGroup, ButtonGroupSeparator } from "@/components/ui/button-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useFactoryPullRequestMergeability, useMergeFactoryPullRequest } from "@/hooks/useFactoryPullRequestMerge";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import {
  defaultMergeMethod,
  isGitHubPullRequest,
  isMergedPullRequest,
  PULL_REQUEST_REVIEW_COPY,
  type FactoryPullRequestMergeMethodChoice,
} from "./splitRunPullRequestReview";

const MERGE_METHODS: FactoryPullRequestMergeMethodChoice[] = [
  "MERGE_METHOD_SQUASH",
  "MERGE_METHOD_MERGE",
  "MERGE_METHOD_REBASE",
];

export function SplitRunPullRequestMergeAction({
  organizationId,
  factoryId,
  orderId,
  pullRequest,
  canAct,
  compact = false,
}: {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  pullRequest?: FactoriesFactoryPullRequest;
  canAct: boolean;
  compact?: boolean;
}) {
  if (!isGitHubPullRequest(pullRequest) || !pullRequest?.id) {
    return null;
  }
  if (isMergedPullRequest(pullRequest)) {
    return (
      <p className={mergedClassName(compact)} data-testid="split-run-pr-merged">
        {PULL_REQUEST_REVIEW_COPY.merged}
      </p>
    );
  }

  return (
    <ConnectedPullRequestMergeAction
      organizationId={organizationId ?? ""}
      factoryId={factoryId ?? ""}
      orderId={orderId}
      pullRequestId={pullRequest.id}
      canAct={canAct}
      compact={compact}
    />
  );
}

function ConnectedPullRequestMergeAction({
  organizationId,
  factoryId,
  orderId,
  pullRequestId,
  canAct,
  compact,
}: {
  organizationId: string;
  factoryId: string;
  orderId?: string;
  pullRequestId: string;
  canAct: boolean;
  compact: boolean;
}) {
  const enabled = Boolean(organizationId && factoryId && pullRequestId);
  const mergeabilityQuery = useFactoryPullRequestMergeability(organizationId, factoryId, pullRequestId, { enabled });
  const mergeMutation = useMergeFactoryPullRequest(organizationId, factoryId, orderId);

  return (
    <SplitRunPullRequestMergeControls
      mergeability={mergeabilityQuery.data}
      mergeabilityFailed={mergeabilityQuery.isError}
      merging={mergeMutation.isPending || mergeMutation.isSuccess}
      canAct={canAct}
      compact={compact}
      onMerge={(mergeMethod, expectedHeadSha) => {
        mergeMutation.mutate(
          { pullRequestId, mergeMethod, expectedHeadSha },
          {
            onError: (error) => {
              showErrorToast(getApiErrorMessage(error, "Failed to merge the pull request"));
            },
          },
        );
      }}
    />
  );
}

function allowedMergeMethods(
  methods: FactoryPullRequestMergeabilityMergeMethod[] | undefined,
): FactoryPullRequestMergeMethodChoice[] {
  return (methods ?? []).filter((method): method is FactoryPullRequestMergeMethodChoice =>
    MERGE_METHODS.includes(method as FactoryPullRequestMergeMethodChoice),
  );
}

function selectedMergeMethod(
  allowedMethods: FactoryPullRequestMergeMethodChoice[],
  selectedMethod: FactoryPullRequestMergeMethodChoice | undefined,
) {
  if (selectedMethod && allowedMethods.includes(selectedMethod)) {
    return selectedMethod;
  }
  return defaultMergeMethod(allowedMethods);
}

function mergeabilityIsKnown(
  mergeability: FactoriesFactoryPullRequestMergeability | undefined,
  mergeabilityFailed: boolean,
) {
  return mergeability != null && !mergeabilityFailed;
}

function pullRequestCanMerge(
  canAct: boolean,
  mergeability: FactoriesFactoryPullRequestMergeability | undefined,
  mergeabilityFailed: boolean,
  method: FactoryPullRequestMergeMethodChoice | undefined,
) {
  return Boolean(
    canAct &&
      mergeabilityIsKnown(mergeability, mergeabilityFailed) &&
      mergeability?.canMerge &&
      method &&
      mergeability.headSha,
  );
}

function mergeBlockedMessage(
  mergeability: FactoriesFactoryPullRequestMergeability | undefined,
  mergeabilityFailed: boolean,
) {
  if (mergeabilityIsKnown(mergeability, mergeabilityFailed) && mergeability?.canMerge) {
    return undefined;
  }
  return mergeability?.message;
}

function mergeControlLabel(merging: boolean) {
  if (merging) {
    return PULL_REQUEST_REVIEW_COPY.merging;
  }
  return PULL_REQUEST_REVIEW_COPY.merge;
}

export function SplitRunPullRequestMergeControls({
  mergeability,
  mergeabilityFailed = false,
  merging,
  canAct,
  compact,
  onMerge,
}: {
  mergeability?: FactoriesFactoryPullRequestMergeability;
  mergeabilityFailed?: boolean;
  merging: boolean;
  canAct: boolean;
  compact: boolean;
  onMerge: (mergeMethod: FactoryPullRequestMergeMethodChoice, expectedHeadSha: string) => void;
}) {
  const allowedMethods = allowedMergeMethods(mergeability?.allowedMethods);
  const [selectedMethod, setSelectedMethod] = useState<FactoryPullRequestMergeMethodChoice | undefined>();
  const method = selectedMergeMethod(allowedMethods, selectedMethod);
  const canMerge = pullRequestCanMerge(canAct, mergeability, mergeabilityFailed, method);
  const reason = mergeBlockedMessage(mergeability, mergeabilityFailed);

  useEffect(() => {
    if (method && method !== selectedMethod) {
      setSelectedMethod(method);
    }
  }, [method, selectedMethod]);

  const mergeWith = (chosen: FactoryPullRequestMergeMethodChoice | undefined) => {
    if (!canMerge || !chosen || !mergeability?.headSha || merging) {
      return;
    }
    onMerge(chosen, mergeability.headSha);
  };

  const mergeControls = (
    <ButtonGroup>
      <Button
        type="button"
        variant="outline"
        size={compact ? "sm" : "lg"}
        className={compact ? undefined : "h-11 px-5 text-[15px] font-semibold"}
        disabled={!canMerge || merging}
        onClick={() => mergeWith(method)}
        data-testid="split-run-merge-button"
      >
        {mergeControlLabel(merging)}
      </Button>
      <ButtonGroupSeparator />
      <MergeMethodMenu
        allowedMethods={allowedMethods}
        method={method}
        compact={compact}
        disabled={!canMerge || merging}
        onSelect={(chosen) => {
          setSelectedMethod(chosen);
          mergeWith(chosen);
        }}
      />
    </ButtonGroup>
  );

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <PermissionTooltip allowed={canAct} message={PULL_REQUEST_REVIEW_COPY.permission}>
        {canAct && reason ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="inline-flex" data-testid="split-run-merge-reason">
                {mergeControls}
              </div>
            </TooltipTrigger>
            <TooltipContent side="top">{reason}</TooltipContent>
          </Tooltip>
        ) : (
          mergeControls
        )}
      </PermissionTooltip>
    </div>
  );
}

function MergeMethodMenu({
  allowedMethods,
  method,
  compact,
  disabled,
  onSelect,
}: {
  allowedMethods: FactoryPullRequestMergeMethodChoice[];
  method?: FactoryPullRequestMergeMethodChoice;
  compact: boolean;
  disabled: boolean;
  onSelect: (method: FactoryPullRequestMergeMethodChoice) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size={compact ? "sm" : "lg"}
          className={compact ? "px-2" : "h-11 px-2.5"}
          disabled={disabled}
          aria-label={PULL_REQUEST_REVIEW_COPY.mergeMethod}
          data-testid="split-run-merge-method"
        >
          <ChevronDown className={compact ? "size-3.5" : "size-4"} aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-44">
        {allowedMethods.map((allowed) => (
          <DropdownMenuItem
            key={allowed}
            onSelect={() => onSelect(allowed)}
            data-testid={`split-run-merge-method-${allowed}`}
          >
            {method === allowed ? "✓ " : ""}
            {PULL_REQUEST_REVIEW_COPY.methods[allowed]}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function mergedClassName(compact: boolean) {
  return compact ? "text-[12px] leading-4 text-foreground/70" : "text-[13px] leading-5 text-foreground/70";
}
