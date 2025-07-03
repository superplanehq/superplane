import React, { ReactNode, Component, ReactElement, FC } from 'react';
import { API_ProviderData, API_IframeRenderer, Addon_Types, Addon_TypesEnum, Addon_Collection, Addon_TypesMapping, API_Panels, API_StateMerger, API_Provider, API_Notification, StoryId, API_Settings, API_LoadedRefData, API_PreparedStoryIndex, API_ViewMode, API_StatusState, API_FilterFunction, API_HashEntry, API_LeafEntry, API_StoryEntry, Args, API_IndexHash, API_ComposedRef, API_StatusUpdate, API_DocsEntry, API_Refs, API_SetRefData, API_ComposedRefUpdate, API_Layout, API_UI, API_PanelPositions, API_Versions, API_UnknownEntries, API_Version, Globals, GlobalTypes, Addon_BaseType, Addon_SidebarTopType, Addon_SidebarBottomType, Addon_PageType, Addon_WrapperType, Addon_Config, API_OptionsData, Parameters, ArgTypes } from '@storybook/types';
export { Addon_Type as Addon, API_ComponentEntry as ComponentEntry, API_ComposedRef as ComposedRef, API_DocsEntry as DocsEntry, API_GroupEntry as GroupEntry, API_HashEntry as HashEntry, API_IndexHash as IndexHash, API_LeafEntry as LeafEntry, API_Refs as Refs, API_RootEntry as RootEntry, API_IndexHash as StoriesHash, API_StoryEntry as StoryEntry } from '@storybook/types';
import { RouterData, NavigateOptions } from '@storybook/router';
import { Listener, Channel } from '@storybook/channels';
export { Listener as ChannelListener } from '@storybook/channels';
import { toId } from '@storybook/csf';
import { ThemeVars } from '@storybook/theming';
import { WhatsNewData } from '@storybook/core-events';

type GetState = () => State;
type SetState = (a: any, b: any) => any;
interface Upstream {
    getState: GetState;
    setState: SetState;
}
type Patch = Partial<State>;
type InputFnPatch = (s: State) => Patch;
type InputPatch = Patch | InputFnPatch;
interface Options {
    persistence: 'none' | 'session' | string;
}
type CallBack = (s: State) => void;
declare class Store {
    upstreamGetState: GetState;
    upstreamSetState: SetState;
    constructor({ setState, getState }: Upstream);
    getInitialState(base: State): any;
    getState(): State;
    setState(inputPatch: InputPatch, options?: Options): Promise<State>;
    setState(inputPatch: InputPatch, callback?: CallBack, options?: Options): Promise<State>;
}

type ModuleFn<APIType = unknown, StateType = unknown> = (m: ModuleArgs, options?: any) => {
    init?: () => void | Promise<void>;
    api: APIType;
    state: StateType;
};
type ModuleArgs = RouterData & API_ProviderData<API> & {
    mode?: 'production' | 'development';
    state: State;
    fullAPI: API;
    store: Store;
};

interface SubAPI$c {
    renderPreview?: API_IframeRenderer;
}

interface SubAPI$b {
    /**
     * Returns a collection of elements of a specific type.
     * @protected This is used internally in storybook's manager.
     * @template T - The type of the elements in the collection.
     * @param {Addon_Types | Addon_TypesEnum.experimental_PAGE} type - The type of the elements to retrieve.
     * @returns {API_Collection<T>} - A collection of elements of the specified type.
     */
    getElements: <T extends Addon_Types | Addon_TypesEnum.experimental_PAGE | Addon_TypesEnum.experimental_SIDEBAR_BOTTOM | Addon_TypesEnum.experimental_SIDEBAR_TOP = Addon_Types>(type: T) => Addon_Collection<Addon_TypesMapping[T]>;
    /**
     * Returns a collection of all panels.
     * This is the same as calling getElements('panel')
     * @protected This is used internally in storybook's manager.
     * @deprecated please use getElements('panel') instead. This API will be removed in storybook 8.0.
     * @returns {API_Panels} - A collection of all panels.
     */
    getPanels: () => API_Panels;
    /**
     * Returns a collection of panels currently enabled for the selected story.
     * @protected This is used internally in storybook's manager.
     * @deprecated please use getElements('panel') instead, and do the filtering manually. This API will be removed in storybook 8.0.
     * @returns {API_Panels} - A collection of all panels.
     */
    getStoryPanels: () => API_Panels;
    /**
     * Returns the id of the currently selected panel.
     * @returns {string} - The ID of the currently selected panel.
     */
    getSelectedPanel: () => string;
    /**
     * Sets the currently selected panel via it's ID.
     * @param {string} panelName - The ID of the panel to select.
     * @returns {void}
     */
    setSelectedPanel: (panelName: string) => void;
    /**
     * Sets the state of an addon with the given ID.
     * @template S - The type of the addon state.
     * @param {string} addonId - The ID of the addon to set the state for.
     * @param {S | API_StateMerger<S>} newStateOrMerger - The new state to set, or a function which receives the current state and returns the new state.
     * @param {Options} [options] - Optional options for the state update.
     * @deprecated This API might get dropped, if you are using this, please file an issue.
     * @returns {Promise<S>} - A promise that resolves with the new state after it has been set.
     */
    setAddonState<S>(addonId: string, newStateOrMerger: S | API_StateMerger<S>, options?: Options): Promise<S>;
    /**
     * Returns the state of an addon with the given ID.
     * @template S - The type of the addon state.
     * @param {string} addonId - The ID of the addon to get the state for.
     * @deprecated This API might get dropped, if you are using this, please file an issue.
     * @returns {S} - The state of the addon with the given ID.
     */
    getAddonState<S>(addonId: string): S;
}

interface SubAPI$a {
    /**
     * Returns the channel object.
     * @protected Please do not use, it's for internal use only.
     */
    getChannel: () => API_Provider<API>['channel'];
    /**
     * Adds a listener to the channel for the given event type.
     * Returns a function that can be called to remove the listener.
     * @param type - The event type to listen for. If using a core event, import it from `@storybook/core-events`.
     * @param handler - The callback function to be called when the event is emitted.
     * @returns A function that can be called to remove the listener.
     */
    on: (type: string, handler: Listener) => () => void;
    /**
     * Removes a listener from the channel for the given event type.
     * @param type - The event type to remove the listener from. If using a core event, import it from `@storybook/core-events`.
     * @param handler - The callback function to be removed.
     */
    off: (type: string, handler: Listener) => void;
    /**
     * Emits an event on the channel for the given event type.
     * @param type - The event type to emit. If using a core event, import it from `@storybook/core-events`.
     * @param args - The arguments to pass to the event listener.
     */
    emit: (type: string, ...args: any[]) => void;
    /**
     * Adds a one-time listener to the channel for the given event type.
     * @param type - The event type to listen for. If using a core event, import it from `@storybook/core-events`.
     * @param handler - The callback function to be called when the event is emitted.
     */
    once: (type: string, handler: Listener) => void;
    /**
     * Emits an event to collapse all stories in the UI.
     * @deprecated Use `emit(STORIES_COLLAPSE_ALL)` instead. This API will be removed in Storybook 8.0.
     */
    collapseAll: () => void;
    /**
     * Emits an event to expand all stories in the UI.
     * @deprecated Use `emit(STORIES_EXPAND_ALL)` instead. This API will be removed in Storybook 8.0.
     */
    expandAll: () => void;
}

interface SubState$9 {
    notifications: API_Notification[];
}
/**
 * The API for managing notifications.
 */
interface SubAPI$9 {
    /**
     * Adds a new notification to the list of notifications.
     * If a notification with the same ID already exists, it will be replaced.
     * @param notification - The notification to add.
     */
    addNotification: (notification: API_Notification) => void;
    /**
     * Removes a notification from the list of notifications and calls the onClear callback.
     * @param id - The ID of the notification to remove.
     */
    clearNotification: (id: string) => void;
}

interface SubAPI$8 {
    storeSelection: () => void;
    retrieveSelection: () => StoryId;
    /**
     * Changes the active settings tab.
     * @param path - The path of the settings page to navigate to. The path NOT should include the `/settings` prefix.
     * @example  changeSettingsTab(`about`).
     */
    changeSettingsTab: (path: string) => void;
    /**
     * Closes the settings screen and returns to the last tracked story or the first story.
     */
    closeSettings: () => void;
    /**
     * Checks if the settings screen is currently active.
     * @returns A boolean indicating whether the settings screen is active.
     */
    isSettingsScreenActive: () => boolean;
    /**
     * Navigates to the specified settings page.
     * @param path - The path of the settings page to navigate to. The path should include the `/settings` prefix.
     * @example  navigateToSettingsPage(`/settings/about`).
     * @deprecated Use `changeSettingsTab` instead.
     */
    navigateToSettingsPage: (path: string) => Promise<void>;
}
interface SubState$8 {
    settings: API_Settings;
}

type Direction = -1 | 1;
type ParameterName = string;
type StoryUpdate = Partial<Pick<API_StoryEntry, 'prepared' | 'parameters' | 'initialArgs' | 'argTypes' | 'args'>>;
type DocsUpdate = Partial<Pick<API_DocsEntry, 'prepared' | 'parameters'>>;
interface SubState$7 extends API_LoadedRefData {
    storyId: StoryId;
    internal_index?: API_PreparedStoryIndex;
    viewMode: API_ViewMode;
    status: API_StatusState;
    filters: Record<string, API_FilterFunction>;
}
interface SubAPI$7 {
    /**
     * The `storyId` method is a reference to the `toId` function from `@storybook/csf`, which is used to generate a unique ID for a story.
     * This ID is used to identify a specific story in the Storybook index.
     *
     * @type {typeof toId}
     */
    storyId: typeof toId;
    /**
     * Resolves a story, docs, component or group ID to its corresponding hash entry in the index.
     *
     * @param {StoryId} storyId - The ID of the story to resolve.
     * @param {string} [refsId] - The ID of the refs to use for resolving the story.
     * @returns {API_HashEntry} - The hash entry corresponding to the given story ID.
     */
    resolveStory: (storyId: StoryId, refsId?: string) => API_HashEntry;
    /**
     * Selects the first story to display in the Storybook UI.
     *
     * @returns {void}
     */
    selectFirstStory: () => void;
    /**
     * Selects a story to display in the Storybook UI.
     *
     * @param {string} [kindOrId] - The kind or ID of the story to select.
     * @param {StoryId} [story] - The ID of the story to select.
     * @param {Object} [obj] - An optional object containing additional options.
     * @param {string} [obj.ref] - The ref ID of the story to select.
     * @param {API_ViewMode} [obj.viewMode] - The view mode to display the story in.
     * @returns {void}
     */
    selectStory: (kindOrId?: string, story?: StoryId, obj?: {
        ref?: string;
        viewMode?: API_ViewMode;
    }) => void;
    /**
     * Returns the current story's data, including its ID, kind, name, and parameters.
     *
     * @returns {API_LeafEntry} The current story's data.
     */
    getCurrentStoryData: () => API_LeafEntry;
    /**
     * Sets the prepared story index to the given value.
     *
     * @param {API_PreparedStoryIndex} index - The prepared story index to set.
     * @returns {Promise<void>} A promise that resolves when the prepared story index has been set.
     */
    setIndex: (index: API_PreparedStoryIndex) => Promise<void>;
    /**
     * Jumps to the next or previous component in the index.
     *
     * @param {Direction} direction - The direction to jump. Use -1 to jump to the previous component, and 1 to jump to the next component.
     * @returns {void}
     */
    jumpToComponent: (direction: Direction) => void;
    /**
     * Jumps to the next or previous story in the story index.
     *
     * @param {Direction} direction - The direction to jump. Use -1 to jump to the previous story, and 1 to jump to the next story.
     * @returns {void}
     */
    jumpToStory: (direction: Direction) => void;
    /**
     * Returns the data for the given story ID and optional ref ID.
     *
     * @param {StoryId} storyId - The ID of the story to retrieve data for.
     * @param {string} [refId] - The ID of the ref to retrieve data for. If not provided, retrieves data for the default ref.
     * @returns {API_LeafEntry} The data for the given story ID and optional ref ID.
     */
    getData: (storyId: StoryId, refId?: string) => API_LeafEntry;
    /**
     * Returns a boolean indicating whether the given story ID and optional ref ID have been prepared.
     *
     * @param {StoryId} storyId - The ID of the story to check.
     * @param {string} [refId] - The ID of the ref to check. If not provided, checks all refs for the given story ID.
     * @returns {boolean} A boolean indicating whether the given story ID and optional ref ID have been prepared.
     */
    isPrepared: (storyId: StoryId, refId?: string) => boolean;
    /**
     * Returns the parameters for the given story ID and optional ref ID.
     *
     * @param {StoryId | { storyId: StoryId; refId: string }} storyId - The ID of the story to retrieve parameters for, or an object containing the story ID and ref ID.
     * @param {ParameterName} [parameterName] - The name of the parameter to retrieve. If not provided, returns all parameters.
     * @returns {API_StoryEntry['parameters'] | any} The parameters for the given story ID and optional ref ID.
     */
    getParameters: (storyId: StoryId | {
        storyId: StoryId;
        refId: string;
    }, parameterName?: ParameterName) => API_StoryEntry['parameters'] | any;
    /**
     * Returns the current value of the specified parameter for the currently selected story.
     *
     * @template S - The type of the parameter value.
     * @param {ParameterName} [parameterName] - The name of the parameter to retrieve. If not provided, returns all parameters.
     * @returns {S} The value of the specified parameter for the currently selected story.
     */
    getCurrentParameter<S>(parameterName?: ParameterName): S;
    /**
     * Updates the arguments for the given story with the provided new arguments.
     *
     * @param {API_StoryEntry} story - The story to update the arguments for.
     * @param {Args} newArgs - The new arguments to set for the story.
     * @returns {void}
     */
    updateStoryArgs(story: API_StoryEntry, newArgs: Args): void;
    /**
     * Resets the arguments for the given story to their initial values.
     *
     * @param {API_StoryEntry} story - The story to reset the arguments for.
     * @param {string[]} [argNames] - An optional array of argument names to reset. If not provided, all arguments will be reset.
     * @returns {void}
     */
    resetStoryArgs: (story: API_StoryEntry, argNames?: string[]) => void;
    /**
     * Finds the leaf entry for the given story ID in the given story index.
     *
     * @param {API_IndexHash} index - The story index to search for the leaf entry in.
     * @param {StoryId} storyId - The ID of the story to find the leaf entry for.
     * @returns {API_LeafEntry} The leaf entry for the given story ID, or null if no leaf entry was found.
     */
    findLeafEntry(index: API_IndexHash, storyId: StoryId): API_LeafEntry;
    /**
     * Finds the leaf story ID for the given component or group ID in the given index.
     *
     * @param {API_IndexHash} index - The story index to search for the leaf story ID in.
     * @param {StoryId} storyId - The ID of the story to find the leaf story ID for.
     * @returns {StoryId} The ID of the leaf story, or null if no leaf story was found.
     */
    findLeafStoryId(index: API_IndexHash, storyId: StoryId): StoryId;
    /**
     * Finds the ID of the sibling story in the given direction for the given story ID in the given story index.
     *
     * @param {StoryId} storyId - The ID of the story to find the sibling of.
     * @param {API_IndexHash} index - The story index to search for the sibling in.
     * @param {Direction} direction - The direction to search for the sibling in.
     * @param {boolean} toSiblingGroup - When true, skips over leafs within the same group.
     * @returns {StoryId} The ID of the sibling story, or null if no sibling was found.
     */
    findSiblingStoryId(storyId: StoryId, index: API_IndexHash, direction: Direction, toSiblingGroup: boolean): StoryId;
    /**
     * Fetches the story index from the server.
     *
     * @returns {Promise<void>} A promise that resolves when the index has been fetched.
     */
    fetchIndex: () => Promise<void>;
    /**
     * Updates the story with the given ID with the provided update object.
     *
     * @param {StoryId} storyId - The ID of the story to update.
     * @param {StoryUpdate} update - An object containing the updated story information.
     * @param {API_ComposedRef} [ref] - The composed ref of the story to update.
     * @returns {Promise<void>} A promise that resolves when the story has been updated.
     */
    updateStory: (storyId: StoryId, update: StoryUpdate, ref?: API_ComposedRef) => Promise<void>;
    /**
     * Updates the documentation for the given story ID with the given update object.
     *
     * @param {StoryId} storyId - The ID of the story to update.
     * @param {DocsUpdate} update - An object containing the updated documentation information.
     * @param {API_ComposedRef} [ref] - The composed ref of the story to update.
     * @returns {Promise<void>} A promise that resolves when the documentation has been updated.
     */
    updateDocs: (storyId: StoryId, update: DocsUpdate, ref?: API_ComposedRef) => Promise<void>;
    /**
     * Sets the preview as initialized.
     *
     * @param {ComposedRef} [ref] - The composed ref of the story to set as initialized.
     * @returns {Promise<void>} A promise that resolves when the preview has been set as initialized.
     */
    setPreviewInitialized: (ref?: API_ComposedRef) => Promise<void>;
    /**
     * Updates the status of a collection of stories.
     *
     * @param {string} addonId - The ID of the addon to update.
     * @param {StatusUpdate} update - An object containing the updated status information.
     * @returns {Promise<void>} A promise that resolves when the status has been updated.
     */
    experimental_updateStatus: (addonId: string, update: API_StatusUpdate | ((state: API_StatusState) => API_StatusUpdate)) => Promise<void>;
    /**
     * Updates the filtering of the index.
     *
     * @param {string} addonId - The ID of the addon to update.
     * @param {API_FilterFunction} filterFunction - A function that returns a boolean based on the story, index and status.
     * @returns {Promise<void>} A promise that resolves when the state has been updated.
     */
    experimental_setFilter: (addonId: string, filterFunction: API_FilterFunction) => Promise<void>;
}

interface SubState$6 {
    refs: API_Refs;
}
interface SubAPI$6 {
    /**
     * Finds a composed ref by its source.
     * @param {string} source - The source/URL of the composed ref.
     * @returns {API_ComposedRef} - The composed ref object.
     */
    findRef: (source: string) => API_ComposedRef;
    /**
     * Sets a composed ref by its ID and data.
     * @param {string} id - The ID of the composed ref.
     * @param {API_SetRefData} data - The data to set for the composed ref.
     * @param {boolean} [ready] - Whether the composed ref is ready.
     */
    setRef: (id: string, data: API_SetRefData, ready?: boolean) => void;
    /**
     * Updates a composed ref by its ID and update object.
     * @param {string} id - The ID of the composed ref.
     * @param {API_ComposedRefUpdate} ref - The update object for the composed ref.
     */
    updateRef: (id: string, ref: API_ComposedRefUpdate) => void;
    /**
     * Gets all composed refs.
     * @returns {API_Refs} - The composed refs object.
     */
    getRefs: () => API_Refs;
    /**
     * Checks if a composed ref is valid.
     * @param {API_SetRefData} ref - The composed ref to check.
     * @returns {Promise<void>} - A promise that resolves when the check is complete.
     */
    checkRef: (ref: API_SetRefData) => Promise<void>;
    /**
     * Changes the version of a composed ref by its ID and URL.
     * @param {string} id - The ID of the composed ref.
     * @param {string} url - The new URL for the composed ref.
     */
    changeRefVersion: (id: string, url: string) => void;
    /**
     * Changes the state of a composed ref by its ID and previewInitialized flag.
     * @param {string} id - The ID of the composed ref.
     * @param {boolean} previewInitialized - The new previewInitialized flag for the composed ref.
     */
    changeRefState: (id: string, previewInitialized: boolean) => void;
}

interface SubState$5 {
    layout: API_Layout;
    ui: API_UI;
    selectedPanel: string | undefined;
    theme: ThemeVars;
}
interface SubAPI$5 {
    /**
     * Toggles the fullscreen mode of the Storybook UI.
     * @param toggled - Optional boolean value to set the fullscreen mode to. If not provided, it will toggle the current state.
     */
    toggleFullscreen: (toggled?: boolean) => void;
    /**
     * Toggles the visibility of the panel in the Storybook UI.
     * @param toggled - Optional boolean value to set the panel visibility to. If not provided, it will toggle the current state.
     */
    togglePanel: (toggled?: boolean) => void;
    /**
     * Toggles the position of the panel in the Storybook UI.
     * @param position - Optional string value to set the panel position to. If not provided, it will toggle between 'bottom' and 'right'.
     */
    togglePanelPosition: (position?: API_PanelPositions) => void;
    /**
     * Toggles the visibility of the navigation bar in the Storybook UI.
     * @param toggled - Optional boolean value to set the navigation bar visibility to. If not provided, it will toggle the current state.
     */
    toggleNav: (toggled?: boolean) => void;
    /**
     * Toggles the visibility of the toolbar in the Storybook UI.
     * @param toggled - Optional boolean value to set the toolbar visibility to. If not provided, it will toggle the current state.
     */
    toggleToolbar: (toggled?: boolean) => void;
    /**
     * Sets the options for the Storybook UI.
     * @param options - An object containing the options to set.
     */
    setOptions: (options: any) => void;
}

declare const isMacLike: () => boolean;
declare const controlOrMetaSymbol: () => "⌘" | "ctrl";
declare const controlOrMetaKey: () => "meta" | "control";
declare const optionOrAltSymbol: () => "⌥" | "alt";
declare const isShortcutTaken: (arr1: string[], arr2: string[]) => boolean;
type KeyboardEventLike = Pick<KeyboardEvent, 'altKey' | 'ctrlKey' | 'metaKey' | 'shiftKey' | 'key' | 'code' | 'keyCode' | 'preventDefault'>;
declare const eventToShortcut: (e: KeyboardEventLike) => (string | string[])[] | null;
declare const shortcutMatchesShortcut: (inputShortcut: (string | string[])[], shortcut: API_KeyCollection) => boolean;
declare const eventMatchesShortcut: (e: KeyboardEventLike, shortcut: API_KeyCollection) => boolean;
declare const keyToSymbol: (key: string) => string;
declare const shortcutToHumanString: (shortcut: API_KeyCollection) => string;

interface SubState$4 {
    shortcuts: API_Shortcuts;
}
interface SubAPI$4 {
    /**
     * Returns the current shortcuts.
     */
    getShortcutKeys(): API_Shortcuts;
    /**
     * Returns the default shortcuts.
     */
    getDefaultShortcuts(): API_Shortcuts | API_AddonShortcutDefaults;
    /**
     * Returns the shortcuts for addons.
     */
    getAddonsShortcuts(): API_AddonShortcuts;
    /**
     * Returns the labels for addon shortcuts.
     */
    getAddonsShortcutLabels(): API_AddonShortcutLabels;
    /**
     * Returns the default shortcuts for addons.
     */
    getAddonsShortcutDefaults(): API_AddonShortcutDefaults;
    /**
     * Sets the shortcuts to the given value.
     * @param shortcuts The new shortcuts to set.
     * @returns A promise that resolves to the new shortcuts.
     */
    setShortcuts(shortcuts: API_Shortcuts): Promise<API_Shortcuts>;
    /**
     * Sets the shortcut for the given action to the given value.
     * @param action The action to set the shortcut for.
     * @param value The new shortcut to set.
     * @returns A promise that resolves to the new shortcut.
     */
    setShortcut(action: API_Action, value: API_KeyCollection): Promise<API_KeyCollection>;
    /**
     * Sets the shortcut for the given addon to the given value.
     * @param addon The addon to set the shortcut for.
     * @param shortcut The new shortcut to set.
     * @returns A promise that resolves to the new addon shortcut.
     */
    setAddonShortcut(addon: string, shortcut: API_AddonShortcut): Promise<API_AddonShortcut>;
    /**
     * Restores all default shortcuts.
     * @returns A promise that resolves to the new shortcuts.
     */
    restoreAllDefaultShortcuts(): Promise<API_Shortcuts>;
    /**
     * Restores the default shortcut for the given action.
     * @param action The action to restore the default shortcut for.
     * @returns A promise that resolves to the new shortcut.
     */
    restoreDefaultShortcut(action: API_Action): Promise<API_KeyCollection>;
    /**
     * Handles a keydown event.
     * @param event The event to handle.
     */
    handleKeydownEvent(event: KeyboardEventLike): void;
    /**
     * Handles a shortcut feature.
     * @param feature The feature to handle.
     * @param event The event to handle.
     */
    handleShortcutFeature(feature: API_Action, event: KeyboardEventLike): void;
}
type API_KeyCollection = string[];
interface API_Shortcuts {
    fullScreen: API_KeyCollection;
    togglePanel: API_KeyCollection;
    panelPosition: API_KeyCollection;
    toggleNav: API_KeyCollection;
    toolbar: API_KeyCollection;
    search: API_KeyCollection;
    focusNav: API_KeyCollection;
    focusIframe: API_KeyCollection;
    focusPanel: API_KeyCollection;
    prevComponent: API_KeyCollection;
    nextComponent: API_KeyCollection;
    prevStory: API_KeyCollection;
    nextStory: API_KeyCollection;
    shortcutsPage: API_KeyCollection;
    aboutPage: API_KeyCollection;
    escape: API_KeyCollection;
    collapseAll: API_KeyCollection;
    expandAll: API_KeyCollection;
    remount: API_KeyCollection;
}
type API_Action = keyof API_Shortcuts;
interface API_AddonShortcut {
    label: string;
    defaultShortcut: API_KeyCollection;
    actionName: string;
    showInMenu?: boolean;
    action: (...args: any[]) => any;
}
type API_AddonShortcuts = Record<string, API_AddonShortcut>;
type API_AddonShortcutLabels = Record<string, string>;
type API_AddonShortcutDefaults = Record<string, API_KeyCollection>;

interface SubState$3 {
    customQueryParams: QueryParams;
}
interface QueryParams {
    [key: string]: string | null;
}
/**
 * SubAPI for managing URL navigation and state.
 */
interface SubAPI$3 {
    /**
     * Navigate to a new URL.
     * @param {string} url - The URL to navigate to.
     * @param {NavigateOptions} options - Options for the navigation.
     * @returns {void}
     */
    navigateUrl: (url: string, options: NavigateOptions) => void;
    /**
     * Get the value of a query parameter from the current URL.
     * @param {string} key - The key of the query parameter to get.
     * @returns {string | undefined} The value of the query parameter, or undefined if it does not exist.
     */
    getQueryParam: (key: string) => string | undefined;
    /**
     * Returns an object containing the current state of the URL.
     * @returns {{
     *   queryParams: QueryParams,
     *   path: string,
     *   viewMode?: string,
     *   storyId?: string,
     *   url: string
     * }} An object containing the current state of the URL.
     */
    getUrlState: () => {
        queryParams: QueryParams;
        path: string;
        viewMode?: string;
        storyId?: string;
        url: string;
    };
    /**
     * Set the query parameters for the current URL.
     * @param {QueryParams} input - An object containing the query parameters to set.
     * @returns {void}
     */
    setQueryParams: (input: QueryParams) => void;
}

interface SubState$2 {
    versions: API_Versions & API_UnknownEntries;
    lastVersionCheck: number;
    dismissedVersionNotification: undefined | string;
}
interface SubAPI$2 {
    /**
     * Returns the current version of the Storybook Manager.
     *
     * @returns {API_Version} The current version of the Storybook Manager.
     */
    getCurrentVersion: () => API_Version;
    /**
     * Returns the latest version of the Storybook Manager.
     *
     * @returns {API_Version} The latest version of the Storybook Manager.
     */
    getLatestVersion: () => API_Version;
    /**
     * Checks if an update is available for the Storybook Manager.
     *
     * @returns {boolean} True if an update is available, false otherwise.
     */
    versionUpdateAvailable: () => boolean;
}

type SubState$1 = {
    whatsNewData?: WhatsNewData;
};
type SubAPI$1 = {
    isWhatsNewUnread(): boolean;
    whatsNewHasBeenRead(): void;
    toggleWhatsNewNotifications(): void;
};

interface SubState {
    globals?: Globals;
    globalTypes?: GlobalTypes;
}
interface SubAPI {
    /**
     * Returns the current global data object.
     * @returns {Globals} The current global data object.
     */
    getGlobals: () => Globals;
    /**
     * Returns the current global types object.
     * @returns {GlobalTypes} The current global types object.
     */
    getGlobalTypes: () => GlobalTypes;
    /**
     * Updates the current global data object with the provided new global data object.
     * @param {Globals} newGlobals - The new global data object to update with.
     * @returns {void}
     */
    updateGlobals: (newGlobals: Globals) => void;
}

declare function mockChannel(): Channel;

interface DeprecatedAddonWithId {
    /**
     * @deprecated will be removed in 8.0, when registering addons, please use the addon id as the first argument
     */
    id?: string;
}
declare class AddonStore {
    constructor();
    private loaders;
    private elements;
    private config;
    private channel;
    /**
     * @deprecated will be removed in 8.0
     */
    private serverChannel;
    private promise;
    private resolve;
    getChannel: () => Channel;
    /**
     * @deprecated will be removed in 8.0, use getChannel instead
     */
    getServerChannel: () => Channel;
    ready: () => Promise<Channel>;
    hasChannel: () => boolean;
    /**
     * @deprecated will be removed in 8.0, please use the normal channel instead
     */
    hasServerChannel: () => boolean;
    setChannel: (channel: Channel) => void;
    /**
     * @deprecated will be removed in 8.0, please use the normal channel instead
     */
    setServerChannel: (channel: Channel) => void;
    getElements<T extends Addon_Types | Addon_TypesEnum.experimental_PAGE | Addon_TypesEnum.experimental_SIDEBAR_BOTTOM | Addon_TypesEnum.experimental_SIDEBAR_TOP>(type: T): Addon_Collection<Addon_TypesMapping[T]>;
    /**
     * Adds a panel to the addon store.
     * @param {string} id - The id of the panel.
     * @param {Addon_Type} options - The options for the panel.
     * @returns {void}
     *
     * @deprecated Use the 'add' method instead.
     * @example
     * addons.add('My Panel', {
     *   title: 'My Title',
     *   type: types.PANEL,
     *   render: () => <div>My Content</div>,
     * });
     */
    addPanel: (id: string, options: Omit<Addon_BaseType, 'type' | 'id'> & DeprecatedAddonWithId) => void;
    /**
     * Adds an addon to the addon store.
     * @param {string} id - The id of the addon.
     * @param {Addon_Type} addon - The addon to add.
     * @returns {void}
     */
    add(id: string, addon: Addon_BaseType | (Omit<Addon_SidebarTopType, 'id'> & DeprecatedAddonWithId) | (Omit<Addon_SidebarBottomType, 'id'> & DeprecatedAddonWithId) | (Omit<Addon_PageType, 'id'> & DeprecatedAddonWithId) | (Omit<Addon_WrapperType, 'id'> & DeprecatedAddonWithId)): void;
    setConfig: (value: Addon_Config) => void;
    getConfig: () => Addon_Config;
    /**
     * Registers an addon loader function.
     *
     * @param {string} id - The id of the addon loader.
     * @param {(api: API) => void} callback - The function that will be called to register the addon.
     * @returns {void}
     */
    register: (id: string, callback: (api: API) => void) => void;
    loadAddons: (api: any) => void;
}
declare const addons: AddonStore;

declare const _default: <TObj = any>(a: TObj, b: Partial<TObj>) => TObj & Partial<TObj>;

declare const ActiveTabs: {
    SIDEBAR: "sidebar";
    CANVAS: "canvas";
    ADDONS: "addons";
};

declare const ManagerContext: React.Context<{
    api: API;
    state: State;
}>;
type State = SubState$5 & SubState$7 & SubState$6 & SubState$9 & SubState$2 & SubState$3 & SubState$4 & SubState$8 & SubState & SubState$1 & RouterData & API_OptionsData & DeprecatedState & Other;
type API = SubAPI$b & SubAPI$a & SubAPI$c & SubAPI$7 & SubAPI$6 & SubAPI & SubAPI$5 & SubAPI$9 & SubAPI$4 & SubAPI$8 & SubAPI$2 & SubAPI$3 & SubAPI$1 & Other;
interface DeprecatedState {
    /**
     * @deprecated use index
     */
    storiesHash: API_IndexHash;
    /**
     * @deprecated use previewInitialized
     */
    storiesConfigured: boolean;
    /**
     * @deprecated use indexError
     */
    storiesFailed?: Error;
}
interface Other {
    [key: string]: any;
}
interface Combo {
    api: API;
    state: State;
}
type ManagerProviderProps = RouterData & API_ProviderData<API> & {
    children: ReactNode | ((props: Combo) => ReactNode);
};
declare const combineParameters: (...parameterSets: Parameters[]) => any;
declare class ManagerProvider extends Component<ManagerProviderProps, State> {
    api: API;
    modules: ReturnType<ModuleFn>[];
    static displayName: string;
    constructor(props: ManagerProviderProps);
    static getDerivedStateFromProps(props: ManagerProviderProps, state: State): State;
    shouldComponentUpdate(nextProps: ManagerProviderProps, nextState: State): boolean;
    initModules: () => void;
    render(): React.JSX.Element;
}
interface ManagerConsumerProps<P = unknown> {
    filter?: (combo: Combo) => P;
    children: FC<P> | ReactNode;
}
declare function ManagerConsumer<P = Combo>({ filter, children, }: ManagerConsumerProps<P>): ReactElement;
declare function useStorybookState(): State;
declare function useStorybookApi(): API;

interface API_EventMap {
    [eventId: string]: Listener;
}
declare const useChannel: (eventMap: API_EventMap, deps?: any[]) => (type: string, ...args: any[]) => void;
declare function useStoryPrepared(storyId?: StoryId): boolean;
declare function useParameter<S>(parameterKey: string, defaultValue?: S): S;
declare function useSharedState<S>(stateId: string, defaultState?: S): [S, (newStateOrMerger: S | API_StateMerger<S>, options?: Options) => void];
declare function useAddonState<S>(addonId: string, defaultState?: S): [S, (newStateOrMerger: S | API_StateMerger<S>, options?: Options) => void];
declare function useArgs(): [Args, (newArgs: Args) => void, (argNames?: string[]) => void];
declare function useGlobals(): [Args, (newGlobals: Args) => void];
declare function useGlobalTypes(): ArgTypes;
declare function useArgTypes(): ArgTypes;

/**
 * We need to rename this so it's not compiled to a straight re-export
 * Our globalization plugin can't handle an import and export of the same name in different lines
 * @deprecated
 */
declare const typesX: typeof Addon_TypesEnum;

export { API, API_EventMap, ActiveTabs, AddonStore, Combo, ManagerConsumer as Consumer, KeyboardEventLike, ManagerContext, ManagerProviderProps, ManagerProvider as Provider, State, Options as StoreOptions, addons, combineParameters, controlOrMetaKey, controlOrMetaSymbol, eventMatchesShortcut, eventToShortcut, isMacLike, isShortcutTaken, keyToSymbol, _default as merge, mockChannel, optionOrAltSymbol, shortcutMatchesShortcut, shortcutToHumanString, typesX as types, useAddonState, useArgTypes, useArgs, useChannel, useGlobalTypes, useGlobals, useParameter, useSharedState, useStoryPrepared, useStorybookApi, useStorybookState };
