import type { MeUserApiToken } from "@/api-client/types.gen";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

interface PersonalApiTokensTableProps {
  tokens: MeUserApiToken[];
  isLoading: boolean;
  onCreate: () => void;
  onRevoke: (token: MeUserApiToken) => void;
  className?: string;
}

const headerCellClassName = "border-b border-border pb-2 pr-4 text-xs font-medium text-muted-foreground last:pr-0";
const bodyCellClassName = "border-b border-border py-2.5 pr-4 text-sm last:border-b-0 last:pr-0";

function formatDay(value?: string) {
  if (!value) return null;

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;

  return {
    day: date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" }),
    exact: date.toLocaleString(),
  };
}

function TokenDate({ value }: { value?: string }) {
  const formatted = formatDay(value);

  if (!formatted) {
    return <span className="text-muted-foreground">Never</span>;
  }

  return (
    <span className="text-muted-foreground" title={formatted.exact}>
      {formatted.day}
    </span>
  );
}

export function PersonalApiTokensTable({
  tokens,
  isLoading,
  onCreate,
  onRevoke,
  className,
}: PersonalApiTokensTableProps) {
  if (isLoading) {
    return <p className={cn("text-sm text-muted-foreground", className)}>Loading tokens...</p>;
  }

  if (tokens.length === 0) {
    return (
      <div
        className={cn(
          "flex flex-col items-center rounded-lg border border-dashed border-border px-6 py-10 text-center",
          className,
        )}
        data-testid="user-token-empty"
      >
        <Icon name="key-round" size="lg" className="text-muted-foreground" />
        <p className="mt-2 text-sm font-medium text-foreground">No API tokens</p>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground">
          Create a token to call the SuperPlane API from a script or another tool.
        </p>
        <Button size="sm" className="mt-4" onClick={onCreate} data-testid="user-token-empty-create">
          <Icon name="plus" />
          Create token
        </Button>
      </div>
    );
  }

  return (
    <table className={cn("w-full text-left", className)} data-testid="user-token-list">
      <thead>
        <tr>
          <th scope="col" className={headerCellClassName}>
            Name
          </th>
          <th scope="col" className={headerCellClassName}>
            Created
          </th>
          <th scope="col" className={headerCellClassName}>
            Last used
          </th>
          <th scope="col" className={cn(headerCellClassName, "w-10")}>
            <span className="sr-only">Actions</span>
          </th>
        </tr>
      </thead>
      <tbody>
        {tokens.map((token) => (
          <tr key={token.id} data-testid="user-token-row">
            <td className={cn(bodyCellClassName, "font-medium text-foreground")}>{token.name || "Unnamed"}</td>
            <td className={bodyCellClassName}>
              <TokenDate value={token.createdAt} />
            </td>
            <td className={bodyCellClassName}>
              <TokenDate value={token.lastUsedAt} />
            </td>
            <td className={cn(bodyCellClassName, "text-right")}>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Actions for ${token.name || "Unnamed"}`}
                    data-testid="user-token-row-menu"
                  >
                    <Icon name="ellipsis-vertical" size="sm" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem
                    onClick={() => onRevoke(token)}
                    className="text-red-600 dark:text-red-400"
                    data-testid="user-token-revoke-btn"
                  >
                    <Icon name="trash-2" size="sm" />
                    Revoke token
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
