"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf, __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: !0 });
}, __copyProps = (to, from, except, desc) => {
  if (from && typeof from == "object" || typeof from == "function")
    for (let key of __getOwnPropNames(from))
      !__hasOwnProp.call(to, key) && key !== except && __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: !0 }) : target,
  mod
)), __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: !0 }), mod);

// src/index.ts
var src_exports = {};
__export(src_exports, {
  buildQueries: () => buildQueries,
  configure: () => configure,
  createEvent: () => createEvent,
  findAllByAltText: () => findAllByAltText,
  findAllByDisplayValue: () => findAllByDisplayValue,
  findAllByLabelText: () => findAllByLabelText,
  findAllByPlaceholderText: () => findAllByPlaceholderText,
  findAllByRole: () => findAllByRole,
  findAllByTestId: () => findAllByTestId,
  findAllByText: () => findAllByText,
  findAllByTitle: () => findAllByTitle,
  findByAltText: () => findByAltText,
  findByDisplayValue: () => findByDisplayValue,
  findByLabelText: () => findByLabelText,
  findByPlaceholderText: () => findByPlaceholderText,
  findByRole: () => findByRole,
  findByTestId: () => findByTestId,
  findByText: () => findByText,
  findByTitle: () => findByTitle,
  fireEvent: () => fireEvent,
  getAllByAltText: () => getAllByAltText,
  getAllByDisplayValue: () => getAllByDisplayValue,
  getAllByLabelText: () => getAllByLabelText,
  getAllByPlaceholderText: () => getAllByPlaceholderText,
  getAllByRole: () => getAllByRole,
  getAllByTestId: () => getAllByTestId,
  getAllByText: () => getAllByText,
  getAllByTitle: () => getAllByTitle,
  getByAltText: () => getByAltText,
  getByDisplayValue: () => getByDisplayValue,
  getByLabelText: () => getByLabelText,
  getByPlaceholderText: () => getByPlaceholderText,
  getByRole: () => getByRole,
  getByTestId: () => getByTestId,
  getByText: () => getByText,
  getByTitle: () => getByTitle,
  getConfig: () => getConfig,
  getDefaultNormalizer: () => getDefaultNormalizer,
  getElementError: () => getElementError,
  getNodeText: () => getNodeText,
  getQueriesForElement: () => getQueriesForElement,
  getRoles: () => getRoles,
  getSuggestedQuery: () => getSuggestedQuery,
  isInaccessible: () => isInaccessible,
  logDOM: () => logDOM,
  logRoles: () => logRoles,
  prettyDOM: () => prettyDOM,
  prettyFormat: () => prettyFormat,
  queries: () => queries,
  queryAllByAltText: () => queryAllByAltText,
  queryAllByAttribute: () => queryAllByAttribute,
  queryAllByDisplayValue: () => queryAllByDisplayValue,
  queryAllByLabelText: () => queryAllByLabelText,
  queryAllByPlaceholderText: () => queryAllByPlaceholderText,
  queryAllByRole: () => queryAllByRole,
  queryAllByTestId: () => queryAllByTestId,
  queryAllByText: () => queryAllByText,
  queryAllByTitle: () => queryAllByTitle,
  queryByAltText: () => queryByAltText,
  queryByAttribute: () => queryByAttribute,
  queryByDisplayValue: () => queryByDisplayValue,
  queryByLabelText: () => queryByLabelText,
  queryByPlaceholderText: () => queryByPlaceholderText,
  queryByRole: () => queryByRole,
  queryByTestId: () => queryByTestId,
  queryByText: () => queryByText,
  queryByTitle: () => queryByTitle,
  queryHelpers: () => queryHelpers,
  screen: () => screen,
  userEvent: () => userEvent,
  waitFor: () => waitFor,
  waitForElementToBeRemoved: () => waitForElementToBeRemoved,
  within: () => within
});
module.exports = __toCommonJS(src_exports);

// node_modules/@storybook/global/dist/index.mjs
var scope = (() => {
  let win;
  return typeof window < "u" ? win = window : typeof globalThis < "u" ? win = globalThis : typeof global < "u" ? win = global : typeof self < "u" ? win = self : win = {}, win;
})();

// node_modules/@storybook/client-logger/dist/index.mjs
var { LOGLEVEL } = scope, levels = { trace: 1, debug: 2, info: 3, warn: 4, error: 5, silent: 10 }, currentLogLevelString = LOGLEVEL, currentLogLevelNumber = levels[currentLogLevelString] || levels.info, logger = { trace: (message, ...rest) => {
  currentLogLevelNumber <= levels.trace && console.trace(message, ...rest);
}, debug: (message, ...rest) => {
  currentLogLevelNumber <= levels.debug && console.debug(message, ...rest);
}, info: (message, ...rest) => {
  currentLogLevelNumber <= levels.info && console.info(message, ...rest);
}, warn: (message, ...rest) => {
  currentLogLevelNumber <= levels.warn && console.warn(message, ...rest);
}, error: (message, ...rest) => {
  currentLogLevelNumber <= levels.error && console.error(message, ...rest);
}, log: (message, ...rest) => {
  currentLogLevelNumber < levels.silent && console.log(message, ...rest);
} }, logged = /* @__PURE__ */ new Set(), once = (type) => (message, ...rest) => {
  if (!logged.has(message))
    return logged.add(message), logger[type](message, ...rest);
};
once.clear = () => logged.clear();
once.trace = once("trace");
once.debug = once("debug");
once.info = once("info");
once.warn = once("warn");
once.error = once("error");
once.log = once("log");
var deprecate = once("warn"), pretty = (type) => (...args) => {
  let argArray = [];
  if (args.length) {
    let startTagRe = /<span\s+style=(['"])([^'"]*)\1\s*>/gi, endTagRe = /<\/span>/gi, reResultArray;
    for (argArray.push(args[0].replace(startTagRe, "%c").replace(endTagRe, "%c")); reResultArray = startTagRe.exec(args[0]); )
      argArray.push(reResultArray[2]), argArray.push("");
    for (let j = 1; j < args.length; j++)
      argArray.push(args[j]);
  }
  logger[type].apply(logger, argArray);
};
pretty.trace = pretty("trace");
pretty.debug = pretty("debug");
pretty.info = pretty("info");
pretty.warn = pretty("warn");
pretty.error = pretty("error");

// node_modules/@storybook/channels/dist/chunk-NH5GSF3H.mjs
var isMulti = (args) => args.transports !== void 0, generateRandomId = () => Math.random().toString(16).slice(2), Channel = class {
  constructor(input = {}) {
    this.sender = generateRandomId(), this.events = {}, this.data = {}, this.transports = [], this.isAsync = input.async || !1, isMulti(input) ? (this.transports = input.transports || [], this.transports.forEach((t) => {
      t.setHandler((event) => this.handleEvent(event));
    })) : this.transports = input.transport ? [input.transport] : [], this.transports.forEach((t) => {
      t.setHandler((event) => this.handleEvent(event));
    });
  }
  get hasTransport() {
    return this.transports.length > 0;
  }
  addListener(eventName, listener) {
    this.events[eventName] = this.events[eventName] || [], this.events[eventName].push(listener);
  }
  emit(eventName, ...args) {
    let event = { type: eventName, args, from: this.sender }, options = {};
    args.length >= 1 && args[0] && args[0].options && (options = args[0].options);
    let handler = () => {
      this.transports.forEach((t) => {
        t.send(event, options);
      }), this.handleEvent(event);
    };
    this.isAsync ? setImmediate(handler) : handler();
  }
  last(eventName) {
    return this.data[eventName];
  }
  eventNames() {
    return Object.keys(this.events);
  }
  listenerCount(eventName) {
    let listeners = this.listeners(eventName);
    return listeners ? listeners.length : 0;
  }
  listeners(eventName) {
    return this.events[eventName] || void 0;
  }
  once(eventName, listener) {
    let onceListener = this.onceListener(eventName, listener);
    this.addListener(eventName, onceListener);
  }
  removeAllListeners(eventName) {
    eventName ? this.events[eventName] && delete this.events[eventName] : this.events = {};
  }
  removeListener(eventName, listener) {
    let listeners = this.listeners(eventName);
    listeners && (this.events[eventName] = listeners.filter((l) => l !== listener));
  }
  on(eventName, listener) {
    this.addListener(eventName, listener);
  }
  off(eventName, listener) {
    this.removeListener(eventName, listener);
  }
  handleEvent(event) {
    let listeners = this.listeners(event.type);
    listeners && listeners.length && listeners.forEach((fn) => {
      fn.apply(event, event.args);
    }), this.data[event.type] = event.args;
  }
  onceListener(eventName, listener) {
    let onceListener = (...args) => (this.removeListener(eventName, onceListener), listener(...args));
    return onceListener;
  }
};

// node_modules/@storybook/core-events/dist/index.mjs
var events = ((events2) => (events2.CHANNEL_CREATED = "channelCreated", events2.CONFIG_ERROR = "configError", events2.STORY_INDEX_INVALIDATED = "storyIndexInvalidated", events2.STORY_SPECIFIED = "storySpecified", events2.SET_CONFIG = "setConfig", events2.SET_STORIES = "setStories", events2.SET_INDEX = "setIndex", events2.SET_CURRENT_STORY = "setCurrentStory", events2.CURRENT_STORY_WAS_SET = "currentStoryWasSet", events2.FORCE_RE_RENDER = "forceReRender", events2.FORCE_REMOUNT = "forceRemount", events2.PRELOAD_ENTRIES = "preloadStories", events2.STORY_PREPARED = "storyPrepared", events2.DOCS_PREPARED = "docsPrepared", events2.STORY_CHANGED = "storyChanged", events2.STORY_UNCHANGED = "storyUnchanged", events2.STORY_RENDERED = "storyRendered", events2.STORY_MISSING = "storyMissing", events2.STORY_ERRORED = "storyErrored", events2.STORY_THREW_EXCEPTION = "storyThrewException", events2.STORY_RENDER_PHASE_CHANGED = "storyRenderPhaseChanged", events2.PLAY_FUNCTION_THREW_EXCEPTION = "playFunctionThrewException", events2.UPDATE_STORY_ARGS = "updateStoryArgs", events2.STORY_ARGS_UPDATED = "storyArgsUpdated", events2.RESET_STORY_ARGS = "resetStoryArgs", events2.SET_GLOBALS = "setGlobals", events2.UPDATE_GLOBALS = "updateGlobals", events2.GLOBALS_UPDATED = "globalsUpdated", events2.REGISTER_SUBSCRIPTION = "registerSubscription", events2.PREVIEW_KEYDOWN = "previewKeydown", events2.PREVIEW_BUILDER_PROGRESS = "preview_builder_progress", events2.SELECT_STORY = "selectStory", events2.STORIES_COLLAPSE_ALL = "storiesCollapseAll", events2.STORIES_EXPAND_ALL = "storiesExpandAll", events2.DOCS_RENDERED = "docsRendered", events2.SHARED_STATE_CHANGED = "sharedStateChanged", events2.SHARED_STATE_SET = "sharedStateSet", events2.NAVIGATE_URL = "navigateUrl", events2.UPDATE_QUERY_PARAMS = "updateQueryParams", events2.REQUEST_WHATS_NEW_DATA = "requestWhatsNewData", events2.RESULT_WHATS_NEW_DATA = "resultWhatsNewData", events2.SET_WHATS_NEW_CACHE = "setWhatsNewCache", events2.TOGGLE_WHATS_NEW_NOTIFICATIONS = "toggleWhatsNewNotifications", events2.TELEMETRY_ERROR = "telemetryError", events2))(events || {});
var { CHANNEL_CREATED, CONFIG_ERROR, CURRENT_STORY_WAS_SET, DOCS_PREPARED, DOCS_RENDERED, FORCE_RE_RENDER, FORCE_REMOUNT, GLOBALS_UPDATED, NAVIGATE_URL, PLAY_FUNCTION_THREW_EXCEPTION, PRELOAD_ENTRIES, PREVIEW_BUILDER_PROGRESS, PREVIEW_KEYDOWN, REGISTER_SUBSCRIPTION, RESET_STORY_ARGS, SELECT_STORY, SET_CONFIG, SET_CURRENT_STORY, SET_GLOBALS, SET_INDEX, SET_STORIES, SHARED_STATE_CHANGED, SHARED_STATE_SET, STORIES_COLLAPSE_ALL, STORIES_EXPAND_ALL, STORY_ARGS_UPDATED, STORY_CHANGED, STORY_ERRORED, STORY_INDEX_INVALIDATED, STORY_MISSING, STORY_PREPARED, STORY_RENDER_PHASE_CHANGED, STORY_RENDERED, STORY_SPECIFIED, STORY_THREW_EXCEPTION, STORY_UNCHANGED, UPDATE_GLOBALS, UPDATE_QUERY_PARAMS, UPDATE_STORY_ARGS, REQUEST_WHATS_NEW_DATA, RESULT_WHATS_NEW_DATA, SET_WHATS_NEW_CACHE, TOGGLE_WHATS_NEW_NOTIFICATIONS, TELEMETRY_ERROR } = events, IGNORED_EXCEPTION = new Error("ignoredException");

// node_modules/@storybook/channels/dist/index.mjs
var { CONFIG_TYPE } = scope;

// node_modules/@storybook/preview-api/dist/chunk-2WNKQWTL.mjs
function mockChannel() {
  let transport = { setHandler: () => {
  }, send: () => {
  } };
  return new Channel({ transport });
}
var AddonStore = class {
  constructor() {
    this.getChannel = () => {
      if (!this.channel) {
        let channel = mockChannel();
        return this.setChannel(channel), channel;
      }
      return this.channel;
    }, this.getServerChannel = () => {
      if (!this.serverChannel)
        throw new Error("Accessing non-existent serverChannel");
      return this.serverChannel;
    }, this.ready = () => this.promise, this.hasChannel = () => !!this.channel, this.hasServerChannel = () => !!this.serverChannel, this.setChannel = (channel) => {
      this.channel = channel, this.resolve();
    }, this.setServerChannel = (channel) => {
      this.serverChannel = channel;
    }, this.promise = new Promise((res) => {
      this.resolve = () => res(this.getChannel());
    });
  }
}, KEY = "__STORYBOOK_ADDONS_PREVIEW";
function getAddonsStore() {
  return scope[KEY] || (scope[KEY] = new AddonStore()), scope[KEY];
}
var addons = getAddonsStore();

// node_modules/@storybook/instrumenter/dist/index.mjs
var CallStates = ((CallStates2) => (CallStates2.DONE = "done", CallStates2.ERROR = "error", CallStates2.ACTIVE = "active", CallStates2.WAITING = "waiting", CallStates2))(CallStates || {}), EVENTS = { CALL: "storybook/instrumenter/call", SYNC: "storybook/instrumenter/sync", START: "storybook/instrumenter/start", BACK: "storybook/instrumenter/back", GOTO: "storybook/instrumenter/goto", NEXT: "storybook/instrumenter/next", END: "storybook/instrumenter/end" }, controlsDisabled = { start: !1, back: !1, goto: !1, next: !1, end: !1 }, alreadyCompletedException = new Error("This function ran after the play function completed. Did you forget to `await` it?"), isObject = (o) => Object.prototype.toString.call(o) === "[object Object]", isModule = (o) => Object.prototype.toString.call(o) === "[object Module]", isInstrumentable = (o) => {
  if (!isObject(o) && !isModule(o))
    return !1;
  if (o.constructor === void 0)
    return !0;
  let proto = o.constructor.prototype;
  return !(!isObject(proto) || Object.prototype.hasOwnProperty.call(proto, "isPrototypeOf") === !1);
}, construct = (obj) => {
  try {
    return new obj.constructor();
  } catch {
    return {};
  }
}, getInitialState = () => ({ renderPhase: void 0, isDebugging: !1, isPlaying: !1, isLocked: !1, cursor: 0, calls: [], shadowCalls: [], callRefsByResult: /* @__PURE__ */ new Map(), chainedCallIds: /* @__PURE__ */ new Set(), ancestors: [], playUntil: void 0, resolvers: {}, syncTimeout: void 0 }), getRetainedState = (state, isDebugging = !1) => {
  let calls = (isDebugging ? state.shadowCalls : state.calls).filter((call) => call.retain);
  if (!calls.length)
    return;
  let callRefsByResult = new Map(Array.from(state.callRefsByResult.entries()).filter(([, ref]) => ref.retain));
  return { cursor: calls.length, calls, callRefsByResult };
}, Instrumenter = class {
  constructor() {
    this.initialized = !1, this.channel = addons.getChannel(), this.state = scope.window.parent.__STORYBOOK_ADDON_INTERACTIONS_INSTRUMENTER_STATE__ || {};
    let resetState = ({ storyId, isPlaying = !0, isDebugging = !1 }) => {
      let state = this.getState(storyId);
      this.setState(storyId, { ...getInitialState(), ...getRetainedState(state, isDebugging), shadowCalls: isDebugging ? state.shadowCalls : [], chainedCallIds: isDebugging ? state.chainedCallIds : /* @__PURE__ */ new Set(), playUntil: isDebugging ? state.playUntil : void 0, isPlaying, isDebugging }), this.sync(storyId);
    };
    this.channel.on(FORCE_REMOUNT, resetState), this.channel.on(STORY_RENDER_PHASE_CHANGED, ({ storyId, newPhase }) => {
      let { isDebugging } = this.getState(storyId);
      this.setState(storyId, { renderPhase: newPhase }), newPhase === "preparing" && isDebugging && resetState({ storyId }), newPhase === "playing" && resetState({ storyId, isDebugging }), newPhase === "played" && this.setState(storyId, { isLocked: !1, isPlaying: !1, isDebugging: !1 }), newPhase === "errored" && this.setState(storyId, { isLocked: !1, isPlaying: !1 });
    }), this.channel.on(SET_CURRENT_STORY, () => {
      this.initialized ? this.cleanup() : this.initialized = !0;
    });
    let start = ({ storyId, playUntil }) => {
      this.getState(storyId).isDebugging || this.setState(storyId, ({ calls }) => ({ calls: [], shadowCalls: calls.map((call) => ({ ...call, status: "waiting" })), isDebugging: !0 }));
      let log = this.getLog(storyId);
      this.setState(storyId, ({ shadowCalls }) => {
        if (playUntil || !log.length)
          return { playUntil };
        let firstRowIndex = shadowCalls.findIndex((call) => call.id === log[0].callId);
        return { playUntil: shadowCalls.slice(0, firstRowIndex).filter((call) => call.interceptable && !call.ancestors.length).slice(-1)[0]?.id };
      }), this.channel.emit(FORCE_REMOUNT, { storyId, isDebugging: !0 });
    }, back = ({ storyId }) => {
      let log = this.getLog(storyId).filter((call) => !call.ancestors.length), last = log.reduceRight((res, item, index) => res >= 0 || item.status === "waiting" ? res : index, -1);
      start({ storyId, playUntil: log[last - 1]?.callId });
    }, goto = ({ storyId, callId }) => {
      let { calls, shadowCalls, resolvers } = this.getState(storyId), call = calls.find(({ id }) => id === callId), shadowCall = shadowCalls.find(({ id }) => id === callId);
      if (!call && shadowCall && Object.values(resolvers).length > 0) {
        let nextId = this.getLog(storyId).find((c) => c.status === "waiting")?.callId;
        shadowCall.id !== nextId && this.setState(storyId, { playUntil: shadowCall.id }), Object.values(resolvers).forEach((resolve) => resolve());
      } else
        start({ storyId, playUntil: callId });
    }, next = ({ storyId }) => {
      let { resolvers } = this.getState(storyId);
      if (Object.values(resolvers).length > 0)
        Object.values(resolvers).forEach((resolve) => resolve());
      else {
        let nextId = this.getLog(storyId).find((c) => c.status === "waiting")?.callId;
        nextId ? start({ storyId, playUntil: nextId }) : end({ storyId });
      }
    }, end = ({ storyId }) => {
      this.setState(storyId, { playUntil: void 0, isDebugging: !1 }), Object.values(this.getState(storyId).resolvers).forEach((resolve) => resolve());
    };
    this.channel.on(EVENTS.START, start), this.channel.on(EVENTS.BACK, back), this.channel.on(EVENTS.GOTO, goto), this.channel.on(EVENTS.NEXT, next), this.channel.on(EVENTS.END, end);
  }
  getState(storyId) {
    return this.state[storyId] || getInitialState();
  }
  setState(storyId, update) {
    let state = this.getState(storyId), patch = typeof update == "function" ? update(state) : update;
    this.state = { ...this.state, [storyId]: { ...state, ...patch } }, scope.window.parent.__STORYBOOK_ADDON_INTERACTIONS_INSTRUMENTER_STATE__ = this.state;
  }
  cleanup() {
    this.state = Object.entries(this.state).reduce((acc, [storyId, state]) => {
      let retainedState = getRetainedState(state);
      return retainedState && (acc[storyId] = Object.assign(getInitialState(), retainedState)), acc;
    }, {});
    let payload = { controlStates: controlsDisabled, logItems: [] };
    this.channel.emit(EVENTS.SYNC, payload), scope.window.parent.__STORYBOOK_ADDON_INTERACTIONS_INSTRUMENTER_STATE__ = this.state;
  }
  getLog(storyId) {
    let { calls, shadowCalls } = this.getState(storyId), merged = [...shadowCalls];
    calls.forEach((call, index) => {
      merged[index] = call;
    });
    let seen = /* @__PURE__ */ new Set();
    return merged.reduceRight((acc, call) => (call.args.forEach((arg) => {
      arg?.__callId__ && seen.add(arg.__callId__);
    }), call.path.forEach((node) => {
      node.__callId__ && seen.add(node.__callId__);
    }), (call.interceptable || call.exception) && !seen.has(call.id) && (acc.unshift({ callId: call.id, status: call.status, ancestors: call.ancestors }), seen.add(call.id)), acc), []);
  }
  instrument(obj, options) {
    if (!isInstrumentable(obj))
      return obj;
    let { mutate = !1, path = [] } = options;
    return Object.keys(obj).reduce((acc, key) => {
      let value = obj[key];
      return typeof value != "function" ? (acc[key] = this.instrument(value, { ...options, path: path.concat(key) }), acc) : typeof value.__originalFn__ == "function" ? (acc[key] = value, acc) : (acc[key] = (...args) => this.track(key, value, args, options), acc[key].__originalFn__ = value, Object.defineProperty(acc[key], "name", { value: key, writable: !1 }), Object.keys(value).length > 0 && Object.assign(acc[key], this.instrument({ ...value }, { ...options, path: path.concat(key) })), acc);
    }, mutate ? obj : construct(obj));
  }
  track(method, fn, args, options) {
    let storyId = args?.[0]?.__storyId__ || scope.__STORYBOOK_PREVIEW__?.selectionStore?.selection?.storyId, { cursor, ancestors } = this.getState(storyId);
    this.setState(storyId, { cursor: cursor + 1 });
    let id = `${ancestors.slice(-1)[0] || storyId} [${cursor}] ${method}`, { path = [], intercept = !1, retain = !1 } = options, interceptable = typeof intercept == "function" ? intercept(method, path) : intercept, call = { id, cursor, storyId, ancestors, path, method, args, interceptable, retain }, result = (interceptable && !ancestors.length ? this.intercept : this.invoke).call(this, fn, call, options);
    return this.instrument(result, { ...options, mutate: !0, path: [{ __callId__: call.id }] });
  }
  intercept(fn, call, options) {
    let { chainedCallIds, isDebugging, playUntil } = this.getState(call.storyId), isChainedUpon = chainedCallIds.has(call.id);
    return !isDebugging || isChainedUpon || playUntil ? (playUntil === call.id && this.setState(call.storyId, { playUntil: void 0 }), this.invoke(fn, call, options)) : new Promise((resolve) => {
      this.setState(call.storyId, ({ resolvers }) => ({ isLocked: !1, resolvers: { ...resolvers, [call.id]: resolve } }));
    }).then(() => (this.setState(call.storyId, (state) => {
      let { [call.id]: _, ...resolvers } = state.resolvers;
      return { isLocked: !0, resolvers };
    }), this.invoke(fn, call, options)));
  }
  invoke(fn, call, options) {
    let { callRefsByResult, renderPhase } = this.getState(call.storyId), serializeValues = (value) => {
      if (callRefsByResult.has(value))
        return callRefsByResult.get(value);
      if (value instanceof Array)
        return value.map(serializeValues);
      if (value instanceof Date)
        return { __date__: { value: value.toISOString() } };
      if (value instanceof Error) {
        let { name, message, stack } = value;
        return { __error__: { name, message, stack } };
      }
      if (value instanceof RegExp) {
        let { flags, source } = value;
        return { __regexp__: { flags, source } };
      }
      if (value instanceof scope.window.HTMLElement) {
        let { prefix, localName, id, classList, innerText } = value, classNames = Array.from(classList);
        return { __element__: { prefix, localName, id, classNames, innerText } };
      }
      return typeof value == "function" ? { __function__: { name: value.name } } : typeof value == "symbol" ? { __symbol__: { description: value.description } } : typeof value == "object" && value?.constructor?.name && value?.constructor?.name !== "Object" ? { __class__: { name: value.constructor.name } } : Object.prototype.toString.call(value) === "[object Object]" ? Object.fromEntries(Object.entries(value).map(([key, val]) => [key, serializeValues(val)])) : value;
    }, info = { ...call, args: call.args.map(serializeValues) };
    call.path.forEach((ref) => {
      ref?.__callId__ && this.setState(call.storyId, ({ chainedCallIds }) => ({ chainedCallIds: new Set(Array.from(chainedCallIds).concat(ref.__callId__)) }));
    });
    let handleException = (e) => {
      if (e instanceof Error) {
        let { name, message, stack, callId = call.id } = e, exception = { name, message, stack, callId };
        if (this.update({ ...info, status: "error", exception }), this.setState(call.storyId, (state) => ({ callRefsByResult: new Map([...Array.from(state.callRefsByResult.entries()), [e, { __callId__: call.id, retain: call.retain }]]) })), call.ancestors.length)
          throw Object.prototype.hasOwnProperty.call(e, "callId") || Object.defineProperty(e, "callId", { value: call.id }), e;
        if (e !== alreadyCompletedException)
          throw logger.warn(e), IGNORED_EXCEPTION;
      }
      throw e;
    };
    try {
      if (renderPhase === "played" && !call.retain)
        throw alreadyCompletedException;
      let finalArgs = (options.getArgs ? options.getArgs(call, this.getState(call.storyId)) : call.args).map((arg) => typeof arg != "function" || Object.keys(arg).length ? arg : (...args) => {
        let { cursor, ancestors } = this.getState(call.storyId);
        this.setState(call.storyId, { cursor: 0, ancestors: [...ancestors, call.id] });
        let restore = () => this.setState(call.storyId, { cursor, ancestors }), willRestore = !1;
        try {
          let res = arg(...args);
          return res instanceof Promise ? (willRestore = !0, res.finally(restore)) : res;
        } finally {
          willRestore || restore();
        }
      }), result = fn(...finalArgs);
      return result && ["object", "function", "symbol"].includes(typeof result) && this.setState(call.storyId, (state) => ({ callRefsByResult: new Map([...Array.from(state.callRefsByResult.entries()), [result, { __callId__: call.id, retain: call.retain }]]) })), this.update({ ...info, status: result instanceof Promise ? "active" : "done" }), result instanceof Promise ? result.then((value) => (this.update({ ...info, status: "done" }), value), handleException) : result;
    } catch (e) {
      return handleException(e);
    }
  }
  update(call) {
    this.channel.emit(EVENTS.CALL, call), this.setState(call.storyId, ({ calls }) => {
      let callsById = calls.concat(call).reduce((a, c) => Object.assign(a, { [c.id]: c }), {});
      return { calls: Object.values(callsById).sort((a, b) => a.id.localeCompare(b.id, void 0, { numeric: !0 })) };
    }), this.sync(call.storyId);
  }
  sync(storyId) {
    let synchronize = () => {
      let { isLocked, isPlaying } = this.getState(storyId), logItems = this.getLog(storyId), pausedAt = logItems.filter(({ ancestors }) => !ancestors.length).find((item) => item.status === "waiting")?.callId, hasActive = logItems.some((item) => item.status === "active");
      if (isLocked || hasActive || logItems.length === 0) {
        let payload2 = { controlStates: controlsDisabled, logItems };
        this.channel.emit(EVENTS.SYNC, payload2);
        return;
      }
      let hasPrevious = logItems.some((item) => item.status === "done" || item.status === "error"), payload = { controlStates: { start: hasPrevious, back: hasPrevious, goto: !0, next: isPlaying, end: isPlaying }, logItems, pausedAt };
      this.channel.emit(EVENTS.SYNC, payload);
    };
    this.setState(storyId, ({ syncTimeout }) => (clearTimeout(syncTimeout), { syncTimeout: setTimeout(synchronize, 0) }));
  }
};
function instrument(obj, options = {}) {
  try {
    let forceInstrument = !1, skipInstrument = !1;
    return scope.window.location?.search?.includes("instrument=true") ? forceInstrument = !0 : scope.window.location?.search?.includes("instrument=false") && (skipInstrument = !0), scope.window.parent === scope.window && !forceInstrument || skipInstrument ? obj : (scope.window.__STORYBOOK_ADDON_INTERACTIONS_INSTRUMENTER__ || (scope.window.__STORYBOOK_ADDON_INTERACTIONS_INSTRUMENTER__ = new Instrumenter()), scope.window.__STORYBOOK_ADDON_INTERACTIONS_INSTRUMENTER__.instrument(obj, options));
  } catch (e) {
    return once.warn(e), obj;
  }
}

// src/index.ts
var domTestingLibrary = __toESM(require("@testing-library/dom")), import_user_event = __toESM(require("@testing-library/user-event")), import_ts_dedent = __toESM(require("ts-dedent")), _userEvent = import_user_event.default.default || import_user_event.default, testingLibrary = instrument(
  { ...domTestingLibrary },
  {
    intercept: (method, path) => path[0] === "fireEvent" || method.startsWith("findBy") || method.startsWith("waitFor")
  }
);
testingLibrary.screen = Object.entries(testingLibrary.screen).reduce(
  (acc, [key, val]) => Object.defineProperty(acc, key, {
    get() {
      return once.warn(import_ts_dedent.default`
          You are using Testing Library's \`screen\` object. Use \`within(canvasElement)\` instead.
          More info: https://storybook.js.org/docs/react/essentials/interactions
        `), val;
    }
  }),
  { ...testingLibrary.screen }
);
var {
  buildQueries,
  configure,
  createEvent,
  findAllByAltText,
  findAllByDisplayValue,
  findAllByLabelText,
  findAllByPlaceholderText,
  findAllByRole,
  findAllByTestId,
  findAllByText,
  findAllByTitle,
  findByAltText,
  findByDisplayValue,
  findByLabelText,
  findByPlaceholderText,
  findByRole,
  findByTestId,
  findByText,
  findByTitle,
  fireEvent,
  getAllByAltText,
  getAllByDisplayValue,
  getAllByLabelText,
  getAllByPlaceholderText,
  getAllByRole,
  getAllByTestId,
  getAllByText,
  getAllByTitle,
  getByAltText,
  getByDisplayValue,
  getByLabelText,
  getByPlaceholderText,
  getByRole,
  getByTestId,
  getByText,
  getByTitle,
  getConfig,
  getDefaultNormalizer,
  getElementError,
  getNodeText,
  getQueriesForElement,
  getRoles,
  getSuggestedQuery,
  isInaccessible,
  logDOM,
  logRoles,
  prettyDOM,
  queries,
  queryAllByAltText,
  queryAllByAttribute,
  queryAllByDisplayValue,
  queryAllByLabelText,
  queryAllByPlaceholderText,
  queryAllByRole,
  queryAllByTestId,
  queryAllByText,
  queryAllByTitle,
  queryByAltText,
  queryByAttribute,
  queryByDisplayValue,
  queryByLabelText,
  queryByPlaceholderText,
  queryByRole,
  queryByTestId,
  queryByText,
  queryByTitle,
  queryHelpers,
  screen,
  waitFor,
  waitForElementToBeRemoved,
  within,
  prettyFormat
} = testingLibrary, userEvent = instrument(
  { userEvent: _userEvent },
  { intercept: !0 }
).userEvent;
