import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";

interface InstallPageHeaderProps {
  title: string;
  description?: string;
}

export function InstallPageHeader({ title, description }: InstallPageHeaderProps) {
  return (
    <div className="mb-6">
      <Heading level={2} className="!text-2xl mb-1">
        {title}
      </Heading>
      {description ? <Text className="text-gray-800 dark:text-gray-400">{description}</Text> : null}
    </div>
  );
}
