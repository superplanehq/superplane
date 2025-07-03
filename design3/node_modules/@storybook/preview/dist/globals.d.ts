declare const globalsNameReferenceMap: {
    readonly '@storybook/addons': "__STORYBOOK_MODULE_ADDONS__";
    readonly '@storybook/global': "__STORYBOOK_MODULE_GLOBAL__";
    readonly '@storybook/channel-postmessage': "__STORYBOOK_MODULE_CHANNEL_POSTMESSAGE__";
    readonly '@storybook/channel-websocket': "__STORYBOOK_MODULE_CHANNEL_WEBSOCKET__";
    readonly '@storybook/channels': "__STORYBOOK_MODULE_CHANNELS__";
    readonly '@storybook/client-api': "__STORYBOOK_MODULE_CLIENT_API__";
    readonly '@storybook/client-logger': "__STORYBOOK_MODULE_CLIENT_LOGGER__";
    readonly '@storybook/core-client': "__STORYBOOK_MODULE_CORE_CLIENT__";
    readonly '@storybook/core-events': "__STORYBOOK_MODULE_CORE_EVENTS__";
    readonly '@storybook/preview-web': "__STORYBOOK_MODULE_PREVIEW_WEB__";
    readonly '@storybook/preview-api': "__STORYBOOK_MODULE_PREVIEW_API__";
    readonly '@storybook/store': "__STORYBOOK_MODULE_STORE__";
    readonly '@storybook/types': "__STORYBOOK_MODULE_TYPES__";
};
declare const globalPackages: ("@storybook/addons" | "@storybook/global" | "@storybook/channel-postmessage" | "@storybook/channel-websocket" | "@storybook/channels" | "@storybook/client-api" | "@storybook/client-logger" | "@storybook/core-client" | "@storybook/core-events" | "@storybook/preview-web" | "@storybook/preview-api" | "@storybook/store" | "@storybook/types")[];

export { globalPackages, globalsNameReferenceMap };
