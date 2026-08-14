import { Heading } from "@/components/Heading/heading";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "../../factoryPageLayoutStyles";

interface WorkspaceSettingsSectionProps {
  title: string;
  description: string;
  children: ReactNode;
}

export function WorkspaceSettingsSection({ title, description, children }: WorkspaceSettingsSectionProps) {
  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            {title}
          </Heading>
          <p className={cn("mt-1 max-w-2xl", factoryPageSubtitleClassName)}>{description}</p>
        </div>
      </header>
      <div className={factoryContentBodyClassName}>{children}</div>
    </>
  );
}
