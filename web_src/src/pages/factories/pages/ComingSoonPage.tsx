import { Heading } from "@/components/Heading/heading";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";

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
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            {title}
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>{description}</p>
        </div>
      </header>
      <div className={factoryContentBodyClassName}>
        <div
          className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border bg-card px-8 py-16 text-center"
          data-testid="coming-soon-body"
        >
          <Icon className="h-10 w-10 text-muted-foreground" aria-hidden />
          <p className="mt-4 text-[15px] font-semibold text-foreground">Soon</p>
          <p className="mt-2 max-w-md text-[13px] text-muted-foreground">
            This section is coming soon. Content for this page comes next.
          </p>
        </div>
      </div>
    </>
  );
}
