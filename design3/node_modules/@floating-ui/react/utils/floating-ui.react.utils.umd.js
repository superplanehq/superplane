(function (global, factory) {
  typeof exports === 'object' && typeof module !== 'undefined' ? factory(exports) :
  typeof define === 'function' && define.amd ? define(['exports'], factory) :
  (global = typeof globalThis !== 'undefined' ? globalThis : global || self, factory(global.FloatingUIReactUtils = {}));
})(this, (function (exports) { 'use strict';

  function hasWindow() {
    return typeof window !== 'undefined';
  }
  function getWindow(node) {
    var _node$ownerDocument;
    return (node == null || (_node$ownerDocument = node.ownerDocument) == null ? void 0 : _node$ownerDocument.defaultView) || window;
  }
  function isHTMLElement(value) {
    if (!hasWindow()) {
      return false;
    }
    return value instanceof HTMLElement || value instanceof getWindow(value).HTMLElement;
  }
  function isShadowRoot(value) {
    if (!hasWindow() || typeof ShadowRoot === 'undefined') {
      return false;
    }
    return value instanceof ShadowRoot || value instanceof getWindow(value).ShadowRoot;
  }

  function activeElement(doc) {
    let activeElement = doc.activeElement;
    while (((_activeElement = activeElement) == null || (_activeElement = _activeElement.shadowRoot) == null ? void 0 : _activeElement.activeElement) != null) {
      var _activeElement;
      activeElement = activeElement.shadowRoot.activeElement;
    }
    return activeElement;
  }
  function contains(parent, child) {
    if (!parent || !child) {
      return false;
    }
    const rootNode = child.getRootNode == null ? void 0 : child.getRootNode();

    // First, attempt with faster native method
    if (parent.contains(child)) {
      return true;
    }

    // then fallback to custom implementation with Shadow DOM support
    if (rootNode && isShadowRoot(rootNode)) {
      let next = child;
      while (next) {
        if (parent === next) {
          return true;
        }
        // @ts-ignore
        next = next.parentNode || next.host;
      }
    }

    // Give up, the result is false
    return false;
  }
  // Avoid Chrome DevTools blue warning.
  function getPlatform() {
    const uaData = navigator.userAgentData;
    if (uaData != null && uaData.platform) {
      return uaData.platform;
    }
    return navigator.platform;
  }
  function getUserAgent() {
    const uaData = navigator.userAgentData;
    if (uaData && Array.isArray(uaData.brands)) {
      return uaData.brands.map(_ref => {
        let {
          brand,
          version
        } = _ref;
        return brand + "/" + version;
      }).join(' ');
    }
    return navigator.userAgent;
  }

  // License: https://github.com/adobe/react-spectrum/blob/b35d5c02fe900badccd0cf1a8f23bb593419f238/packages/@react-aria/utils/src/isVirtualEvent.ts
  function isVirtualClick(event) {
    // FIXME: Firefox is now emitting a deprecation warning for `mozInputSource`.
    // Try to find a workaround for this. `react-aria` source still has the check.
    if (event.mozInputSource === 0 && event.isTrusted) {
      return true;
    }
    if (isAndroid() && event.pointerType) {
      return event.type === 'click' && event.buttons === 1;
    }
    return event.detail === 0 && !event.pointerType;
  }
  function isVirtualPointerEvent(event) {
    if (isJSDOM()) return false;
    return !isAndroid() && event.width === 0 && event.height === 0 || isAndroid() && event.width === 1 && event.height === 1 && event.pressure === 0 && event.detail === 0 && event.pointerType === 'mouse' ||
    // iOS VoiceOver returns 0.333• for width/height.
    event.width < 1 && event.height < 1 && event.pressure === 0 && event.detail === 0 && event.pointerType === 'touch';
  }
  function isSafari() {
    // Chrome DevTools does not complain about navigator.vendor
    return /apple/i.test(navigator.vendor);
  }
  function isAndroid() {
    const re = /android/i;
    return re.test(getPlatform()) || re.test(getUserAgent());
  }
  function isMac() {
    return getPlatform().toLowerCase().startsWith('mac') && !navigator.maxTouchPoints;
  }
  function isJSDOM() {
    return getUserAgent().includes('jsdom/');
  }
  function isMouseLikePointerType(pointerType, strict) {
    // On some Linux machines with Chromium, mouse inputs return a `pointerType`
    // of "pen": https://github.com/floating-ui/floating-ui/issues/2015
    const values = ['mouse', 'pen'];
    if (!strict) {
      values.push('', undefined);
    }
    return values.includes(pointerType);
  }
  function isReactEvent(event) {
    return 'nativeEvent' in event;
  }
  function isRootElement(element) {
    return element.matches('html,body');
  }
  function getDocument(node) {
    return (node == null ? void 0 : node.ownerDocument) || document;
  }
  function isEventTargetWithin(event, node) {
    if (node == null) {
      return false;
    }
    if ('composedPath' in event) {
      return event.composedPath().includes(node);
    }

    // TS thinks `event` is of type never as it assumes all browsers support composedPath, but browsers without shadow dom don't
    const e = event;
    return e.target != null && node.contains(e.target);
  }
  function getTarget(event) {
    if ('composedPath' in event) {
      return event.composedPath()[0];
    }

    // TS thinks `event` is of type never as it assumes all browsers support
    // `composedPath()`, but browsers without shadow DOM don't.
    return event.target;
  }
  const TYPEABLE_SELECTOR = "input:not([type='hidden']):not([disabled])," + "[contenteditable]:not([contenteditable='false']),textarea:not([disabled])";
  function isTypeableElement(element) {
    return isHTMLElement(element) && element.matches(TYPEABLE_SELECTOR);
  }
  function stopEvent(event) {
    event.preventDefault();
    event.stopPropagation();
  }
  function isTypeableCombobox(element) {
    if (!element) return false;
    return element.getAttribute('role') === 'combobox' && isTypeableElement(element);
  }

  exports.TYPEABLE_SELECTOR = TYPEABLE_SELECTOR;
  exports.activeElement = activeElement;
  exports.contains = contains;
  exports.getDocument = getDocument;
  exports.getPlatform = getPlatform;
  exports.getTarget = getTarget;
  exports.getUserAgent = getUserAgent;
  exports.isAndroid = isAndroid;
  exports.isEventTargetWithin = isEventTargetWithin;
  exports.isJSDOM = isJSDOM;
  exports.isMac = isMac;
  exports.isMouseLikePointerType = isMouseLikePointerType;
  exports.isReactEvent = isReactEvent;
  exports.isRootElement = isRootElement;
  exports.isSafari = isSafari;
  exports.isTypeableCombobox = isTypeableCombobox;
  exports.isTypeableElement = isTypeableElement;
  exports.isVirtualClick = isVirtualClick;
  exports.isVirtualPointerEvent = isVirtualPointerEvent;
  exports.stopEvent = stopEvent;

}));
