import { FC } from 'react';

declare global {
	interface SymbolConstructor {
		readonly observable: symbol;
	}
}

type Addon_Types = Exclude<Addon_TypesEnum, Addon_TypesEnum.experimental_PAGE | Addon_TypesEnum.experimental_SIDEBAR_BOTTOM | Addon_TypesEnum.experimental_SIDEBAR_TOP>;
declare enum Addon_TypesEnum {
    /**
     * This API is used to create a tab the toolbar above the canvas, This API might be removed in the future.
     * @unstable
     */
    TAB = "tab",
    /**
     * This adds panels to the addons side panel.
     */
    PANEL = "panel",
    /**
     * This adds items in the toolbar above the canvas - on the left side.
     */
    TOOL = "tool",
    /**
     * This adds items in the toolbar above the canvas - on the right side.
     */
    TOOLEXTRA = "toolextra",
    /**
     * This adds wrapper components around the canvas/iframe component storybook renders.
     * @unstable this API is not stable yet, and is likely to change in 8.0.
     */
    PREVIEW = "preview",
    /**
     * This adds pages that render instead of the canvas.
     * @unstable
     */
    experimental_PAGE = "page",
    /**
     * This adds items in the bottom of the sidebar.
     * @unstable
     */
    experimental_SIDEBAR_BOTTOM = "sidebar-bottom",
    /**
     * This adds items in the top of the sidebar.
     * @unstable This will get replaced with a new API in 8.0, use at your own risk.
     */
    experimental_SIDEBAR_TOP = "sidebar-top",
    /**
     * @deprecated This property does nothing, and will be removed in Storybook 8.0.
     */
    NOTES_ELEMENT = "notes-element"
}

declare class Provider {
    getElements(_type: Addon_Types): void;
    handleAPI(_api: unknown): void;
    getConfig(): {};
}

interface RootProps {
    provider: Provider;
    history?: History;
}
declare const Root: FC<RootProps>;
declare function renderStorybookUI(domNode: HTMLElement, provider: Provider): void;

export { Provider, Root, RootProps, renderStorybookUI };
