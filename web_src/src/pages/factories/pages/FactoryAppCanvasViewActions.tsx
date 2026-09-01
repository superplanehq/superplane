import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { cn } from "@/lib/utils";

type FactoryAppCanvasViewActionsProps = {
  href?: string;
  onOpenVisualEditor?: () => void;
};

const editAutomationClassName = "rounded-md";

export function FactoryAppCanvasViewActions({ href, onOpenVisualEditor }: FactoryAppCanvasViewActionsProps) {
  if (href) {
    return (
      <div className="flex items-start justify-end">
        <Link
          href={href}
          className={cn(buttonVariants({ size: "sm" }), editAutomationClassName)}
          data-testid="factory-app-edit"
        >
          Edit Automation
        </Link>
      </div>
    );
  }
  return (
    <div className="flex items-start justify-end">
      <Button
        type="button"
        size="sm"
        className={editAutomationClassName}
        onClick={onOpenVisualEditor}
        data-testid="factory-app-edit"
      >
        Edit Automation
      </Button>
    </div>
  );
}
