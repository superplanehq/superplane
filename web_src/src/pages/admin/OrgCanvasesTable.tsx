import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Palette, Search } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import AdminPagination from "./AdminPagination";

interface Canvas {
  id: string;
  name: string;
  description: string;
}

interface PaginatedResponse<T> {
  items: T[];
  total: number;
}

const PAGE_SIZE = 50;

export function OrgCanvasesTable({ orgId }: { orgId: string }) {
  const [canvases, setCanvases] = useState<Canvas[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");

  const fetchCanvases = useCallback(
    async (s: string, o: number) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(o) });
      if (s) params.set("search", s);
      const res = await fetch(`/admin/api/organizations/${orgId}/canvases?${params}`, { credentials: "include" });
      if (res.ok) {
        const data: PaginatedResponse<Canvas> = await res.json();
        setCanvases(data.items);
        setTotal(data.total);
      }
    },
    [orgId],
  );

  useEffect(() => {
    const t = setTimeout(() => {
      setOffset(0);
      fetchCanvases(search, 0);
    }, 200);
    return () => clearTimeout(t);
  }, [search, fetchCanvases]);

  return (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-3 gap-3">
        <div className="flex items-center gap-2">
          <Palette size={16} className="text-gray-600 dark:text-gray-400" />
          <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
            Automations ({total})
          </Heading>
        </div>
        <div className="flex items-center gap-3">
          <ResetBuiltInAutomations orgId={orgId} />
          <div className="relative w-56">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500" />
            <input
              type="text"
              placeholder="Search automations..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500"
            />
          </div>
        </div>
      </div>

      {canvases.length === 0 ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">
          {search ? "No automations match your search." : "No automations in this organization."}
        </Text>
      ) : (
        <>
          <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden dark:bg-gray-900 dark:outline-gray-700/70">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100 dark:border-gray-700/70">
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Name</th>
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium dark:text-gray-400">Description</th>
                </tr>
              </thead>
              <tbody>
                {canvases.map((canvas) => (
                  <tr key={canvas.id} className="border-b border-slate-50 last:border-0 dark:border-gray-800/70">
                    <td className="px-4 py-2.5 text-gray-800 font-medium dark:text-gray-100">{canvas.name}</td>
                    <td className="px-4 py-2.5 text-gray-500 dark:text-gray-400">{canvas.description || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <AdminPagination
            offset={offset}
            total={total}
            pageSize={PAGE_SIZE}
            onPageChange={(o) => {
              setOffset(o);
              fetchCanvases(search, o);
            }}
          />
        </>
      )}
    </div>
  );
}

function ResetBuiltInAutomations({ orgId }: { orgId: string }) {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const reset = async () => {
    setPending(true);
    setError(null);
    setMessage(null);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/built-in-automations/reset`, {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        setError("Could not reset built-in automations.");
        return;
      }
      const data: { reset?: number } = await response.json();
      const count = data.reset ?? 0;
      setOpen(false);
      setMessage(resetResultMessage(count));
    } catch {
      setError("Could not reset built-in automations.");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex flex-col items-end gap-1">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => {
          setError(null);
          setOpen(true);
        }}
      >
        Reset built-in automations
      </Button>
      {message ? <Text className="text-xs text-gray-500 dark:text-gray-400">{message}</Text> : null}
      <Dialog open={open} onClose={() => (pending ? undefined : setOpen(false))} size="md">
        <DialogTitle>Reset built-in automations?</DialogTitle>
        <DialogDescription>
          <p>
            This replaces the built-in automations in this organization with the current templates. Repositories and
            integrations stay connected. Custom automations stay unchanged.
          </p>
          {error ? <p className="mt-2 text-red-600 dark:text-red-400">{error}</p> : null}
        </DialogDescription>
        <DialogActions>
          <Button type="button" variant="outline" disabled={pending} onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={pending} onClick={() => void reset()}>
            {pending ? "Resetting..." : "Reset built-in automations"}
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}

function resetResultMessage(count: number): string {
  if (count === 0) {
    return "No built-in automations to reset.";
  }
  if (count === 1) {
    return "Reset 1 built-in automation.";
  }
  return `Reset ${count} built-in automations.`;
}
