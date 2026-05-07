import { Text } from "@/components/Text/text";

export const ErrorBanner = ({ message }: { message: string | null }) => {
  if (!message) {
    return null;
  }

  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-900/20">
      <Text className="text-sm text-red-700 dark:text-red-400">{message}</Text>
    </div>
  );
};
