import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { cn } from "@/lib/utils";

type FactoryAppCanvasViewActionsProps = {
  href?: string;
  onOpenVisualEditor?: () => void;
};

export function FactoryAppCanvasViewActions({ href, onOpenVisualEditor }: FactoryAppCanvasViewActionsProps) {
  if (href) {
    return (
      <div className="flex items-start justify-end">
        <Link href={href} className={cn(buttonVariants({ size: "sm" }))} data-testid="factory-app-edit">
          Edit
        </Link>
      </div>
    );
  }
  return (
    <div className="flex items-start justify-end">
      <Button type="button" size="sm" onClick={onOpenVisualEditor} data-testid="factory-app-edit">
        Edit
      </Button>
    </div>
  );
}
