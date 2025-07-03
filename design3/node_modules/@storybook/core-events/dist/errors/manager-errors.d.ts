import { S as StorybookError } from '../storybook-error-32ac504c.js';

/**
 * If you can't find a suitable category for your error, create one
 * based on the package name/file path of which the error is thrown.
 * For instance:
 * If it's from @storybook/client-logger, then MANAGER_CLIENT-LOGGER
 *
 * Categories are prefixed by a logical grouping, e.g. MANAGER_
 * to prevent manager and preview errors from having the same category and error code.
 */
declare enum Category {
    MANAGER_UNCAUGHT = "MANAGER_UNCAUGHT",
    MANAGER_UI = "MANAGER_UI",
    MANAGER_API = "MANAGER_API",
    MANAGER_CLIENT_LOGGER = "MANAGER_CLIENT-LOGGER",
    MANAGER_CHANNELS = "MANAGER_CHANNELS",
    MANAGER_CORE_EVENTS = "MANAGER_CORE-EVENTS",
    MANAGER_ROUTER = "MANAGER_ROUTER",
    MANAGER_THEMING = "MANAGER_THEMING"
}
declare class ProviderDoesNotExtendBaseProviderError extends StorybookError {
    readonly category = Category.MANAGER_UI;
    readonly code = 1;
    template(): string;
}
declare class UncaughtManagerError extends StorybookError {
    data: {
        error: Error;
    };
    readonly category = Category.MANAGER_UNCAUGHT;
    readonly code = 1;
    constructor(data: {
        error: Error;
    });
    template(): string;
}

export { Category, ProviderDoesNotExtendBaseProviderError, UncaughtManagerError };
