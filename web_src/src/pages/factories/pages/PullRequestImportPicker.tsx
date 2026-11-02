import type { FactoriesFactoryPullRequestImportItem, FactoriesWorkOrder } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useImportFactoryPullRequest, useSearchFactoryPullRequestImports } from "@/hooks/useFactoryPullRequestImports";
import { useAutoLoadMoreOnScroll } from "@/hooks/useAutoLoadMoreOnScroll";
import { FEATURE_FACTORY_PULL_REQUEST_IMPORT } from "@/lib/experimentalFeatures";
import { getApiErrorMessage } from "@/lib/errors";
import { factoryVCSProvider } from "@/pages/home/factories";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { GitPullRequest, Loader2, Plus } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { isCodexActionCancelled } from "../codex/codexActionContext";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { useBacklogCreateConnection } from "./useBacklogCreateConnection";

type PullRequestImportPickerProps = {
  organizationId: string;
  factoryId: string;
  lineId: string;
  mobile?: boolean;
  onOpenTask: (id: string, order?: FactoriesWorkOrder) => void;
};

export function PullRequestImportPicker(props: PullRequestImportPickerProps) {
  const { has } = useExperimentalFeature(props.organizationId);
  const { canAct } = usePermissions();
  if (!has(FEATURE_FACTORY_PULL_REQUEST_IMPORT) || !canAct("work_orders", "create") || !canAct("work_orders", "update"))
    return null;
  return <ConnectedPullRequestImportPicker {...props} />;
}

function ConnectedPullRequestImportPicker({
  organizationId,
  factoryId,
  lineId,
  mobile,
  onOpenTask,
}: PullRequestImportPickerProps) {
  const layout = useOptionalFactoriesLayout();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [limit, setLimit] = useState(5);
  const [error, setError] = useState<string>();
  const pending = useRef(false);
  const prepareOpen = useBacklogCreateConnection(organizationId, factoryId);
  const mutation = useImportFactoryPullRequest(organizationId, factoryId);
  const candidates = useSearchFactoryPullRequestImports(organizationId, factoryId, search, limit, open);
  useEffect(() => {
    const timeout = window.setTimeout(() => {
      setSearch(query);
      setLimit(5);
    }, 300);
    return () => window.clearTimeout(timeout);
  }, [query]);
  const reset = () => {
    setQuery("");
    setSearch("");
    setLimit(5);
    setError(undefined);
  };
  const select = async (item: FactoriesFactoryPullRequestImportItem) => {
    if (pending.current) return;
    if (item.existingTask?.id) {
      onOpenTask(item.existingTask.id, item.existingTask);
      setOpen(false);
      return;
    }
    if (!item.url) return;
    pending.current = true;
    setError(undefined);
    try {
      const order = await mutation.mutateAsync({ lineId, pullRequestUrl: item.url });
      if (order.id) {
        setOpen(false);
        onOpenTask(order.id, order);
      }
    } catch (error) {
      if (!isCodexActionCancelled(error))
        setError(getApiErrorMessage(error, "SuperPlane could not import the pull request. Try again."));
    } finally {
      pending.current = false;
    }
  };
  const opening = useRef(false);
  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        if (pending.current || opening.current) return;
        if (!next) {
          setOpen(false);
          return;
        }
        opening.current = true;
        void prepareOpen()
          .then((ready) => {
            if (ready) {
              reset();
              setOpen(true);
            }
          })
          .finally(() => {
            opening.current = false;
          });
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Import pull request"
          title="Import pull request"
          className={mobile ? "mb-2 size-11 self-end text-muted-foreground" : "size-6 text-muted-foreground"}
        >
          <Plus className="size-3.5" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        side={mobile ? "bottom" : "right"}
        align="start"
        sideOffset={6}
        className={
          mobile ? "w-[calc(100vw-2rem)] p-2 [&_button]:min-h-11 [&_input]:h-11 [&_input]:text-base" : "w-96 p-2"
        }
      >
        <PullRequestImportResults
          vcsProvider={layout?.factory?.onboarding?.vcsProvider}
          items={candidates.data ?? []}
          query={query}
          onQueryChange={setQuery}
          isLoading={candidates.isLoading || query !== search}
          isFetching={candidates.isFetching}
          isPending={mutation.isPending}
          error={
            error ??
            (candidates.isError
              ? getApiErrorMessage(candidates.error, "Check the workspace repository connection and try again.")
              : undefined)
          }
          hasMore={!candidates.isError && limit < 50 && (candidates.data?.length ?? 0) >= limit}
          onLoadMore={() => setLimit((count) => Math.min(50, count + 5))}
          onRetry={() => {
            setError(undefined);
            void candidates.refetch();
          }}
          onSelect={(item) => void select(item)}
        />
      </PopoverContent>
    </Popover>
  );
}

export type PullRequestImportResultsProps = {
  vcsProvider?: string;
  items: FactoriesFactoryPullRequestImportItem[];
  query: string;
  onQueryChange: (query: string) => void;
  isLoading?: boolean;
  isFetching?: boolean;
  isPending?: boolean;
  error?: string;
  hasMore?: boolean;
  onLoadMore: () => void;
  onRetry: () => void;
  onSelect: (item: FactoriesFactoryPullRequestImportItem) => void;
};

export function PullRequestImportResults(props: PullRequestImportResultsProps) {
  const searchId = useId();
  const resultsRef = useRef<HTMLDivElement>(null);
  const loadMore = useAutoLoadMoreOnScroll({
    hasMore: props.hasMore && !props.isPending,
    isLoading: props.isLoading || props.isFetching,
    onLoadMore: props.onLoadMore,
  });
  useEffect(() => {
    loadMore(resultsRef.current);
  }, [props.items.length, props.hasMore, props.isFetching, loadMore]);
  return (
    <>
      <div className="mt-1 flex items-center gap-2.5 px-2.5 py-1.5">
        <GitPullRequest className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        <Label htmlFor={searchId} className="sr-only">
          Search pull requests
        </Label>
        <Input
          id={searchId}
          placeholder={
            factoryVCSProvider(props.vcsProvider) === "bitbucket"
              ? "Import a PR from Bitbucket"
              : "Import a PR from GitHub"
          }
          value={props.query}
          onChange={(event) => props.onQueryChange(event.target.value)}
          disabled={props.isPending}
          className="h-8 px-2.5 text-sm"
        />
      </div>
      {props.error ? (
        <div role="alert" className="p-2 text-sm">
          <p>{props.error}</p>
          <Button variant="ghost" size="sm" disabled={props.isPending} onClick={props.onRetry}>
            Try again
          </Button>
        </div>
      ) : null}
      {props.isPending ? (
        <p role="status" className="flex items-center gap-2 p-2 text-sm">
          <Loader2 className="size-3.5 animate-spin" aria-hidden />
          Importing pull request…
        </p>
      ) : null}
      <div
        ref={resultsRef}
        className="mt-0.5 mb-1 ml-8 flex max-h-44 flex-col overflow-y-auto"
        onScroll={(event) => loadMore(event.currentTarget)}
      >
        {props.isLoading ? (
          <p role="status" className="p-2 text-sm">
            Loading pull requests…
          </p>
        ) : props.items.length === 0 && !props.error ? (
          <p className="p-2 text-sm text-muted-foreground">No open pull requests found. Try a different search.</p>
        ) : (
          props.items.map((item) => (
            <Button
              key={item.url}
              variant="ghost"
              disabled={props.isPending}
              className="h-auto w-full flex-col items-start gap-0 rounded-md px-2.5 py-2 text-left font-normal"
              onClick={() => props.onSelect(item)}
            >
              <span className="flex w-full items-center gap-2">
                <span className="min-w-0 flex-1 truncate">{item.title}</span>
                <span className="text-xs text-muted-foreground">#{item.number}</span>
              </span>
              {item.existingTask ? (
                <span className="text-xs text-muted-foreground">Already imported</span>
              ) : item.draft ? (
                <span className="text-xs text-muted-foreground">Draft</span>
              ) : null}
            </Button>
          ))
        )}
        {props.isFetching && !props.isLoading ? (
          <p role="status" className="p-2 text-sm">
            Loading pull requests…
          </p>
        ) : null}
      </div>
    </>
  );
}
