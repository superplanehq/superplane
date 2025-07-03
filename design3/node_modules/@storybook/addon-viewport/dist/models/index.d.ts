type Styles = ViewportStyles | ((s: ViewportStyles | undefined) => ViewportStyles) | null;
interface Viewport {
    name: string;
    styles: Styles;
    type: 'desktop' | 'mobile' | 'tablet' | 'other';
}
interface ViewportStyles {
    height: string;
    width: string;
}
interface ViewportMap {
    [key: string]: Viewport;
}

interface ViewportAddonParameter {
    disable?: boolean;
    defaultOrientation?: 'portrait' | 'landscape';
    defaultViewport?: string;
    viewports?: ViewportMap;
}

export { Styles, Viewport, ViewportAddonParameter, ViewportMap, ViewportStyles };
