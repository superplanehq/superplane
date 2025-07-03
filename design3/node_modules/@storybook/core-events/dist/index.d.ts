declare enum events {
    CHANNEL_CREATED = "channelCreated",
    CONFIG_ERROR = "configError",
    STORY_INDEX_INVALIDATED = "storyIndexInvalidated",
    STORY_SPECIFIED = "storySpecified",
    SET_CONFIG = "setConfig",
    SET_STORIES = "setStories",
    SET_INDEX = "setIndex",
    SET_CURRENT_STORY = "setCurrentStory",
    CURRENT_STORY_WAS_SET = "currentStoryWasSet",
    FORCE_RE_RENDER = "forceReRender",
    FORCE_REMOUNT = "forceRemount",
    PRELOAD_ENTRIES = "preloadStories",
    STORY_PREPARED = "storyPrepared",
    DOCS_PREPARED = "docsPrepared",
    STORY_CHANGED = "storyChanged",
    STORY_UNCHANGED = "storyUnchanged",
    STORY_RENDERED = "storyRendered",
    STORY_MISSING = "storyMissing",
    STORY_ERRORED = "storyErrored",
    STORY_THREW_EXCEPTION = "storyThrewException",
    STORY_RENDER_PHASE_CHANGED = "storyRenderPhaseChanged",
    PLAY_FUNCTION_THREW_EXCEPTION = "playFunctionThrewException",
    UPDATE_STORY_ARGS = "updateStoryArgs",
    STORY_ARGS_UPDATED = "storyArgsUpdated",
    RESET_STORY_ARGS = "resetStoryArgs",
    SET_GLOBALS = "setGlobals",
    UPDATE_GLOBALS = "updateGlobals",
    GLOBALS_UPDATED = "globalsUpdated",
    REGISTER_SUBSCRIPTION = "registerSubscription",
    PREVIEW_KEYDOWN = "previewKeydown",
    PREVIEW_BUILDER_PROGRESS = "preview_builder_progress",
    SELECT_STORY = "selectStory",
    STORIES_COLLAPSE_ALL = "storiesCollapseAll",
    STORIES_EXPAND_ALL = "storiesExpandAll",
    DOCS_RENDERED = "docsRendered",
    SHARED_STATE_CHANGED = "sharedStateChanged",
    SHARED_STATE_SET = "sharedStateSet",
    NAVIGATE_URL = "navigateUrl",
    UPDATE_QUERY_PARAMS = "updateQueryParams",
    REQUEST_WHATS_NEW_DATA = "requestWhatsNewData",
    RESULT_WHATS_NEW_DATA = "resultWhatsNewData",
    SET_WHATS_NEW_CACHE = "setWhatsNewCache",
    TOGGLE_WHATS_NEW_NOTIFICATIONS = "toggleWhatsNewNotifications",
    TELEMETRY_ERROR = "telemetryError"
}

declare const CHANNEL_CREATED: events;
declare const CONFIG_ERROR: events;
declare const CURRENT_STORY_WAS_SET: events;
declare const DOCS_PREPARED: events;
declare const DOCS_RENDERED: events;
declare const FORCE_RE_RENDER: events;
declare const FORCE_REMOUNT: events;
declare const GLOBALS_UPDATED: events;
declare const NAVIGATE_URL: events;
declare const PLAY_FUNCTION_THREW_EXCEPTION: events;
declare const PRELOAD_ENTRIES: events;
declare const PREVIEW_BUILDER_PROGRESS: events;
declare const PREVIEW_KEYDOWN: events;
declare const REGISTER_SUBSCRIPTION: events;
declare const RESET_STORY_ARGS: events;
declare const SELECT_STORY: events;
declare const SET_CONFIG: events;
declare const SET_CURRENT_STORY: events;
declare const SET_GLOBALS: events;
declare const SET_INDEX: events;
declare const SET_STORIES: events;
declare const SHARED_STATE_CHANGED: events;
declare const SHARED_STATE_SET: events;
declare const STORIES_COLLAPSE_ALL: events;
declare const STORIES_EXPAND_ALL: events;
declare const STORY_ARGS_UPDATED: events;
declare const STORY_CHANGED: events;
declare const STORY_ERRORED: events;
declare const STORY_INDEX_INVALIDATED: events;
declare const STORY_MISSING: events;
declare const STORY_PREPARED: events;
declare const STORY_RENDER_PHASE_CHANGED: events;
declare const STORY_RENDERED: events;
declare const STORY_SPECIFIED: events;
declare const STORY_THREW_EXCEPTION: events;
declare const STORY_UNCHANGED: events;
declare const UPDATE_GLOBALS: events;
declare const UPDATE_QUERY_PARAMS: events;
declare const UPDATE_STORY_ARGS: events;
declare const REQUEST_WHATS_NEW_DATA: events;
declare const RESULT_WHATS_NEW_DATA: events;
declare const SET_WHATS_NEW_CACHE: events;
declare const TOGGLE_WHATS_NEW_NOTIFICATIONS: events;
declare const TELEMETRY_ERROR: events;
declare const IGNORED_EXCEPTION: Error;
interface WhatsNewCache {
    lastDismissedPost?: string;
    lastReadPost?: string;
}
type WhatsNewData = {
    status: 'SUCCESS';
    title: string;
    url: string;
    blogUrl?: string;
    publishedAt: string;
    excerpt: string;
    postIsRead: boolean;
    showNotification: boolean;
    disableWhatsNewNotifications: boolean;
} | {
    status: 'ERROR';
};

export { CHANNEL_CREATED, CONFIG_ERROR, CURRENT_STORY_WAS_SET, DOCS_PREPARED, DOCS_RENDERED, FORCE_REMOUNT, FORCE_RE_RENDER, GLOBALS_UPDATED, IGNORED_EXCEPTION, NAVIGATE_URL, PLAY_FUNCTION_THREW_EXCEPTION, PRELOAD_ENTRIES, PREVIEW_BUILDER_PROGRESS, PREVIEW_KEYDOWN, REGISTER_SUBSCRIPTION, REQUEST_WHATS_NEW_DATA, RESET_STORY_ARGS, RESULT_WHATS_NEW_DATA, SELECT_STORY, SET_CONFIG, SET_CURRENT_STORY, SET_GLOBALS, SET_INDEX, SET_STORIES, SET_WHATS_NEW_CACHE, SHARED_STATE_CHANGED, SHARED_STATE_SET, STORIES_COLLAPSE_ALL, STORIES_EXPAND_ALL, STORY_ARGS_UPDATED, STORY_CHANGED, STORY_ERRORED, STORY_INDEX_INVALIDATED, STORY_MISSING, STORY_PREPARED, STORY_RENDERED, STORY_RENDER_PHASE_CHANGED, STORY_SPECIFIED, STORY_THREW_EXCEPTION, STORY_UNCHANGED, TELEMETRY_ERROR, TOGGLE_WHATS_NEW_NOTIFICATIONS, UPDATE_GLOBALS, UPDATE_QUERY_PARAMS, UPDATE_STORY_ARGS, WhatsNewCache, WhatsNewData, events as default };
