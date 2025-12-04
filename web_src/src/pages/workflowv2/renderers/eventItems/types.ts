export interface EventItemStyleProps {
  iconName: string;
  iconColor: string;
  iconBackground: string;
  titleColor: string;
  iconSize: number;
  iconContainerSize: number;
  iconStrokeWidth: number;
  animation: string;
}

export interface EventItemRenderer {
  getEventItemStyle: (state: string, componentType?: string, eventData?: Record<string, unknown>) => EventItemStyleProps;
}