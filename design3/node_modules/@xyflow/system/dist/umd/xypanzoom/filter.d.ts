export type FilterParams = {
    zoomActivationKeyPressed: boolean;
    zoomOnScroll: boolean;
    zoomOnPinch: boolean;
    panOnDrag: boolean | number[];
    panOnScroll: boolean;
    zoomOnDoubleClick: boolean;
    userSelectionActive: boolean;
    noWheelClassName: string;
    noPanClassName: string;
    lib: string;
};
export declare function createFilter({ zoomActivationKeyPressed, zoomOnScroll, zoomOnPinch, panOnDrag, panOnScroll, zoomOnDoubleClick, userSelectionActive, noWheelClassName, noPanClassName, lib, }: FilterParams): (event: any) => boolean;
//# sourceMappingURL=filter.d.ts.map