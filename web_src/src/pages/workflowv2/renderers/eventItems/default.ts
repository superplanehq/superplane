import { EventItemStyleProps, EventItemRenderer } from "./types";

export const getDefaultEventItemStyle = (state: string): EventItemStyleProps => {
  switch (state) {
    case "processed":
      return {
        iconName: "circle-check",
        iconColor: "text-green-700",
        iconBackground: "bg-green-200",
        titleColor: "text-green-800",
        iconSize: 16,
        iconContainerSize: 4,
        iconStrokeWidth: 2,
        animation: "",
      };
    case "discarded":
      return {
        iconName: "circle-x",
        iconColor: "text-red-700",
        iconBackground: "bg-red-200",
        titleColor: "text-red-800",
        iconSize: 16,
        iconContainerSize: 4,
        iconStrokeWidth: 2,
        animation: "",
      };
    case "waiting":
      return {
        iconName: "circle-dashed",
        iconColor: "text-gray-500",
        iconBackground: "bg-gray-100",
        titleColor: "text-gray-600",
        iconSize: 16,
        iconContainerSize: 4,
        iconStrokeWidth: 2,
        animation: "",
      };
    case "running":
      return {
        iconName: "refresh-cw",
        iconColor: "text-blue-700",
        iconBackground: "bg-blue-100",
        titleColor: "text-blue-800",
        iconSize: 16,
        iconContainerSize: 4,
        iconStrokeWidth: 2,
        animation: "animate-spin",
      };
    default:
      return {
        iconName: "check",
        iconColor: "text-green-700",
        iconBackground: "bg-green-200",
        titleColor: "text-black",
        iconSize: 16,
        iconContainerSize: 4,
        iconStrokeWidth: 2,
        animation: "",
      };
  }
};

export const defaultEventItemRenderer: EventItemRenderer = {
  getEventItemStyle: getDefaultEventItemStyle,
};