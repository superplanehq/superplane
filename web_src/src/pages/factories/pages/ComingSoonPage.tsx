import type { LucideIcon } from "lucide-react";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

interface ComingSoonPageProps {
  title: string;
  description: string;
  Icon: LucideIcon;
}

export function ComingSoonPage({ title, description, Icon }: ComingSoonPageProps) {
  return (
    <>
      <WorkspacePageHeader title={title} subtitle={description} />
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
