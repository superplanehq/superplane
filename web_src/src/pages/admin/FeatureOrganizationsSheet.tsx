import { useEffect, useState } from "react";
import { Link } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/ui/sheet";
import AdminPagination from "./AdminPagination";
import { billingCatalogRequest, type FeatureDefinition } from "./billingPlansApi";

type FeatureOrganizationsPage = {
  items: {
    id: string;
    name: string;
    slug: string;
    enabled_grant: boolean;
    plan: { id: string; key: string; name: string } | null;
  }[];
  total: number;
  limit: number;
  offset: number;
};

export function FeatureOrganizationsSheet({
  feature,
  returnFocus,
  onClose,
}: {
  feature: FeatureDefinition;
  returnFocus: HTMLButtonElement | null;
  onClose: () => void;
}) {
  const [search, setSearch] = useState("");
  const [offset, setOffset] = useState(0);
  const [page, setPage] = useState<FeatureOrganizationsPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(false);
    const query = new URLSearchParams({ search, limit: "50", offset: String(offset) });
    void billingCatalogRequest<FeatureOrganizationsPage>(
      `feature-definitions/${encodeURIComponent(feature.key)}/organizations?${query}`,
    )
      .then((result) => {
        if (active) setPage(result);
      })
      .catch(() => {
        if (active) setError(true);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [feature.key, search, offset, retry]);
  return (
    <Sheet
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent
        overlayClassName="bg-black/20"
        side="right"
        className="flex h-full w-full flex-col gap-0 p-0 sm:w-[640px] sm:max-w-[640px]"
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          returnFocus?.focus();
        }}
      >
        <SheetHeader className="shrink-0 border-b p-6 text-left">
          <SheetTitle className="pr-6 break-words">{feature.label}</SheetTitle>
          <code className="break-all text-xs text-muted-foreground">{feature.key}</code>
          <div>
            <Badge variant="outline">{feature.gated ? "Restricted" : "All organizations"}</Badge>
          </div>
          <SheetDescription>
            {feature.gated
              ? "Organizations with an enabled grant or plan inclusion."
              : "Every organization has access. This list shows explicit grants and plan inclusion."}
          </SheetDescription>
        </SheetHeader>
        <div className="shrink-0 px-6 py-4">
          <Input
            type="search"
            aria-label="Search organizations"
            placeholder="Search by organization name or ID"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value);
              setOffset(0);
            }}
          />
        </div>
        <div className="min-h-0 flex-1 overflow-auto px-6 pb-6" aria-busy={loading}>
          {loading ? (
            <p role="status" className="py-8 text-sm text-muted-foreground">
              Loading organizations...
            </p>
          ) : error ? (
            <div role="alert" className="space-y-3 py-8 text-sm">
              <p>Failed to load organizations.</p>
              <Button variant="outline" onClick={() => setRetry((value) => value + 1)}>
                Try again
              </Button>
            </div>
          ) : (
            page && <FeatureOrganizationsList page={page} search={search} onPageChange={setOffset} />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function FeatureOrganizationsList({
  page,
  search,
  onPageChange,
}: {
  page: FeatureOrganizationsPage;
  search: string;
  onPageChange: (offset: number) => void;
}) {
  return (
    <>
      <Table aria-label="Organizations with feature access">
        <TableHead>
          <TableRow>
            <TableHeader>Organization</TableHeader>
            <TableHeader>Access source</TableHeader>
            <TableHeader>Plan</TableHeader>
          </TableRow>
        </TableHead>
        <TableBody>
          {page.items.length === 0 && (
            <TableRow>
              <TableCell colSpan={3} className="py-8 text-center whitespace-normal text-muted-foreground">
                {search
                  ? "No matching organizations. Try a different search."
                  : "No organizations have an enabled grant or plan inclusion."}
              </TableCell>
            </TableRow>
          )}
          {page.items.map((organization) => (
            <TableRow key={organization.id}>
              <TableCell className="max-w-56 break-words whitespace-normal">
                <Link
                  className="font-medium text-blue-600 hover:underline dark:text-blue-400"
                  to={`/admin/organizations/${organization.id}`}
                >
                  {organization.name}
                </Link>
              </TableCell>
              <TableCell>
                <div className="flex flex-wrap gap-1">
                  {organization.enabled_grant && <Badge variant="outline">Grant</Badge>}
                  {organization.plan && <Badge variant="outline">Plan</Badge>}
                </div>
              </TableCell>
              <TableCell className="max-w-40 break-words whitespace-normal">{organization.plan?.name ?? "—"}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <AdminPagination offset={page.offset} total={page.total} pageSize={page.limit} onPageChange={onPageChange} />
    </>
  );
}
