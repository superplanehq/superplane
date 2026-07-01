import { Text } from "@/components/Text/text";
import { InstallShell } from "./InstallShell";

interface InstallLoadingViewProps {
  message: string;
}

export function InstallLoadingView({ message }: InstallLoadingViewProps) {
  return (
    <InstallShell>
      <div className="flex flex-col items-center justify-center gap-4 py-16">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600" />
        <Text className="text-gray-500 dark:text-gray-400">{message}</Text>
      </div>
    </InstallShell>
  );
}
