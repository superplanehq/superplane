import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";
import React from "react";

interface AdminSearchHeaderProps {
  title: string;
  subtitle: string;
  search: string;
  onSearchChange: (value: string) => void;
  placeholder: string;
  filters?: React.ReactNode;
}

const AdminSearchHeader: React.FC<AdminSearchHeaderProps> = ({
  title,
  subtitle,
  search,
  onSearchChange,
  placeholder,
  filters,
}) => (
  <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
    <div>
      <Heading className="text-gray-800 mb-0.5 dark:text-gray-100">{title}</Heading>
      <Text className="text-gray-500 text-sm dark:text-gray-400">{subtitle}</Text>
    </div>

    <div className="flex flex-wrap items-center gap-2">
      {filters}
      <div className="relative w-72">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500" />
        <Input
          type="text"
          placeholder={placeholder}
          value={search}
          onChange={(event: React.ChangeEvent<HTMLInputElement>) => onSearchChange(event.target.value)}
          className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500"
        />
      </div>
    </div>
  </div>
);

export default AdminSearchHeader;
