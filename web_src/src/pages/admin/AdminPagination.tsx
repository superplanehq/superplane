import { Text } from "@/components/Text/text";
import React from "react";

interface AdminPaginationProps {
  offset: number;
  total: number;
  pageSize: number;
  onPageChange: (newOffset: number) => void;
}

const AdminPagination: React.FC<AdminPaginationProps> = ({ offset, total, pageSize, onPageChange }) => {
  const totalPages = Math.ceil(total / pageSize);
  const currentPage = Math.floor(offset / pageSize) + 1;

  if (totalPages <= 1) return null;

  return (
    <div className="mt-4 flex items-center justify-between text-sm text-content-secondary">
      <Text>
        Showing {offset + 1}–{Math.min(offset + pageSize, total)} of {total}
      </Text>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(offset - pageSize)}
          disabled={offset === 0}
          className="rounded border border-edge-default bg-surface-raised px-3 py-1 text-xs text-content-primary hover:bg-surface-subtle disabled:cursor-not-allowed disabled:opacity-40"
        >
          Previous
        </button>
        <button
          onClick={() => onPageChange(offset + pageSize)}
          disabled={currentPage >= totalPages}
          className="rounded border border-edge-default bg-surface-raised px-3 py-1 text-xs text-content-primary hover:bg-surface-subtle disabled:cursor-not-allowed disabled:opacity-40"
        >
          Next
        </button>
      </div>
    </div>
  );
};

export default AdminPagination;
