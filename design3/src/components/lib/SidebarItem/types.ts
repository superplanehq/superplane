export interface SidebarItemProps {
  title: string;
  subtitle?: string;
  icon?: React.ReactNode;
  onClickAddNode?: () => void;
  onDragStart?: (e: React.DragEvent) => void;
  className?: string;
  comingSoon?: boolean;
  showSubtitle?: boolean;
}