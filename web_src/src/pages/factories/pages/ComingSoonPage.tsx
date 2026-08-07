import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import type { LucideIcon } from "lucide-react";
import { factoryContentBodyClassName, factoryContentHeaderClassName } from "./factoryPageLayoutStyles";

interface ComingSoonPageProps {
  title: string;
  description: string;
  Icon: LucideIcon;
}

export function ComingSoonPage({ title, description, Icon }: ComingSoonPageProps) {
  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className="!text-xl text-gray-900 dark:text-gray-100">
            {title}
          </Heading>
          <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">{description}</Text>
        </div>
      </header>
      <div className={factoryContentBodyClassName}>
        <div
          className="flex flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white px-8 py-16 text-center dark:border-gray-700 dark:bg-gray-900"
          data-testid="coming-soon-body"
        >
          <Icon className="h-10 w-10 text-slate-400 dark:text-gray-500" aria-hidden />
          <p className="mt-4 text-base font-semibold text-slate-900 dark:text-gray-100">Soon</p>
          <Text className="mt-2 max-w-md text-sm text-gray-500 dark:text-gray-400">
            This section is coming soon. Content for this page comes next.
          </Text>
        </div>
      </div>
    </>
  );
}
