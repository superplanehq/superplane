import { S as StorybookError } from '../storybook-error-32ac504c.js';

/**
 * If you can't find a suitable category for your error, create one
 * based on the package name/file path of which the error is thrown.
 * For instance:
 * If it's from @storybook/client-logger, then CLIENT-LOGGER
 *
 * Categories are prefixed by a logical grouping, e.g. PREVIEW_ or FRAMEWORK_
 * to prevent manager and preview errors from having the same category and error code.
 */
declare enum Category {
    PREVIEW_CLIENT_LOGGER = "PREVIEW_CLIENT-LOGGER",
    PREVIEW_CHANNELS = "PREVIEW_CHANNELS",
    PREVIEW_CORE_EVENTS = "PREVIEW_CORE-EVENTS",
    PREVIEW_INSTRUMENTER = "PREVIEW_INSTRUMENTER",
    PREVIEW_API = "PREVIEW_API",
    PREVIEW_REACT_DOM_SHIM = "PREVIEW_REACT-DOM-SHIM",
    PREVIEW_ROUTER = "PREVIEW_ROUTER",
    PREVIEW_THEMING = "PREVIEW_THEMING",
    RENDERER_HTML = "RENDERER_HTML",
    RENDERER_PREACT = "RENDERER_PREACT",
    RENDERER_REACT = "RENDERER_REACT",
    RENDERER_SERVER = "RENDERER_SERVER",
    RENDERER_SVELTE = "RENDERER_SVELTE",
    RENDERER_VUE = "RENDERER_VUE",
    RENDERER_VUE3 = "RENDERER_VUE3",
    RENDERER_WEB_COMPONENTS = "RENDERER_WEB-COMPONENTS"
}
declare class MissingStoryAfterHmrError extends StorybookError {
    data: {
        storyId: string;
    };
    readonly category = Category.PREVIEW_API;
    readonly code = 1;
    constructor(data: {
        storyId: string;
    });
    template(): string;
}
declare class ImplicitActionsDuringRendering extends StorybookError {
    data: {
        phase: string;
        name: string;
        deprecated: boolean;
    };
    readonly category = Category.PREVIEW_API;
    readonly code = 2;
    readonly documentation = "https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#using-implicit-actions-during-rendering-is-deprecated-for-example-in-the-play-function";
    constructor(data: {
        phase: string;
        name: string;
        deprecated: boolean;
    });
    template(): string;
}

export { Category, ImplicitActionsDuringRendering, MissingStoryAfterHmrError };
