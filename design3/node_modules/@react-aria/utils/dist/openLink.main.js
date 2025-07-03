var $1117b6c0d4c4c164$exports = require("./focusWithoutScrolling.main.js");
var $9e20cff0af27e8cc$exports = require("./platform.main.js");
var $6kLG7$react = require("react");


function $parcel$interopDefault(a) {
  return a && a.__esModule ? a.default : a;
}

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "RouterProvider", () => $4068a0fae83b6d84$export$323e4fc2fa4753fb);
$parcel$export(module.exports, "shouldClientNavigate", () => $4068a0fae83b6d84$export$efa8c9099e530235);
$parcel$export(module.exports, "openLink", () => $4068a0fae83b6d84$export$95185d699e05d4d7);
$parcel$export(module.exports, "useRouter", () => $4068a0fae83b6d84$export$9a302a45f65d0572);
$parcel$export(module.exports, "useSyntheticLinkProps", () => $4068a0fae83b6d84$export$bdc77b0c0a3a85d6);
$parcel$export(module.exports, "getSyntheticLinkProps", () => $4068a0fae83b6d84$export$51437d503373d223);
$parcel$export(module.exports, "useLinkProps", () => $4068a0fae83b6d84$export$7e924b3091a3bd18);
/*
 * Copyright 2023 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 


const $4068a0fae83b6d84$var$RouterContext = /*#__PURE__*/ (0, $6kLG7$react.createContext)({
    isNative: true,
    open: $4068a0fae83b6d84$var$openSyntheticLink,
    useHref: (href)=>href
});
function $4068a0fae83b6d84$export$323e4fc2fa4753fb(props) {
    let { children: children, navigate: navigate, useHref: useHref } = props;
    let ctx = (0, $6kLG7$react.useMemo)(()=>({
            isNative: false,
            open: (target, modifiers, href, routerOptions)=>{
                $4068a0fae83b6d84$var$getSyntheticLink(target, (link)=>{
                    if ($4068a0fae83b6d84$export$efa8c9099e530235(link, modifiers)) navigate(href, routerOptions);
                    else $4068a0fae83b6d84$export$95185d699e05d4d7(link, modifiers);
                });
            },
            useHref: useHref || ((href)=>href)
        }), [
        navigate,
        useHref
    ]);
    return /*#__PURE__*/ (0, ($parcel$interopDefault($6kLG7$react))).createElement($4068a0fae83b6d84$var$RouterContext.Provider, {
        value: ctx
    }, children);
}
function $4068a0fae83b6d84$export$9a302a45f65d0572() {
    return (0, $6kLG7$react.useContext)($4068a0fae83b6d84$var$RouterContext);
}
function $4068a0fae83b6d84$export$efa8c9099e530235(link, modifiers) {
    // Use getAttribute here instead of link.target. Firefox will default link.target to "_parent" when inside an iframe.
    let target = link.getAttribute('target');
    return (!target || target === '_self') && link.origin === location.origin && !link.hasAttribute('download') && !modifiers.metaKey && // open in new tab (mac)
    !modifiers.ctrlKey && // open in new tab (windows)
    !modifiers.altKey && // download
    !modifiers.shiftKey;
}
function $4068a0fae83b6d84$export$95185d699e05d4d7(target, modifiers, setOpening = true) {
    var _window_event_type, _window_event;
    let { metaKey: metaKey, ctrlKey: ctrlKey, altKey: altKey, shiftKey: shiftKey } = modifiers;
    // Firefox does not recognize keyboard events as a user action by default, and the popup blocker
    // will prevent links with target="_blank" from opening. However, it does allow the event if the
    // Command/Control key is held, which opens the link in a background tab. This seems like the best we can do.
    // See https://bugzilla.mozilla.org/show_bug.cgi?id=257870 and https://bugzilla.mozilla.org/show_bug.cgi?id=746640.
    if ((0, $9e20cff0af27e8cc$exports.isFirefox)() && ((_window_event = window.event) === null || _window_event === void 0 ? void 0 : (_window_event_type = _window_event.type) === null || _window_event_type === void 0 ? void 0 : _window_event_type.startsWith('key')) && target.target === '_blank') {
        if ((0, $9e20cff0af27e8cc$exports.isMac)()) metaKey = true;
        else ctrlKey = true;
    }
    // WebKit does not support firing click events with modifier keys, but does support keyboard events.
    // https://github.com/WebKit/WebKit/blob/c03d0ac6e6db178f90923a0a63080b5ca210d25f/Source/WebCore/html/HTMLAnchorElement.cpp#L184
    let event = (0, $9e20cff0af27e8cc$exports.isWebKit)() && (0, $9e20cff0af27e8cc$exports.isMac)() && !(0, $9e20cff0af27e8cc$exports.isIPad)() && process.env.NODE_ENV !== 'test' ? new KeyboardEvent('keydown', {
        keyIdentifier: 'Enter',
        metaKey: metaKey,
        ctrlKey: ctrlKey,
        altKey: altKey,
        shiftKey: shiftKey
    }) : new MouseEvent('click', {
        metaKey: metaKey,
        ctrlKey: ctrlKey,
        altKey: altKey,
        shiftKey: shiftKey,
        bubbles: true,
        cancelable: true
    });
    $4068a0fae83b6d84$export$95185d699e05d4d7.isOpening = setOpening;
    (0, $1117b6c0d4c4c164$exports.focusWithoutScrolling)(target);
    target.dispatchEvent(event);
    $4068a0fae83b6d84$export$95185d699e05d4d7.isOpening = false;
}
// https://github.com/parcel-bundler/parcel/issues/8724
$4068a0fae83b6d84$export$95185d699e05d4d7.isOpening = false;
function $4068a0fae83b6d84$var$getSyntheticLink(target, open) {
    if (target instanceof HTMLAnchorElement) open(target);
    else if (target.hasAttribute('data-href')) {
        let link = document.createElement('a');
        link.href = target.getAttribute('data-href');
        if (target.hasAttribute('data-target')) link.target = target.getAttribute('data-target');
        if (target.hasAttribute('data-rel')) link.rel = target.getAttribute('data-rel');
        if (target.hasAttribute('data-download')) link.download = target.getAttribute('data-download');
        if (target.hasAttribute('data-ping')) link.ping = target.getAttribute('data-ping');
        if (target.hasAttribute('data-referrer-policy')) link.referrerPolicy = target.getAttribute('data-referrer-policy');
        target.appendChild(link);
        open(link);
        target.removeChild(link);
    }
}
function $4068a0fae83b6d84$var$openSyntheticLink(target, modifiers) {
    $4068a0fae83b6d84$var$getSyntheticLink(target, (link)=>$4068a0fae83b6d84$export$95185d699e05d4d7(link, modifiers));
}
function $4068a0fae83b6d84$export$bdc77b0c0a3a85d6(props) {
    let router = $4068a0fae83b6d84$export$9a302a45f65d0572();
    var _props_href;
    const href = router.useHref((_props_href = props.href) !== null && _props_href !== void 0 ? _props_href : '');
    return {
        'data-href': props.href ? href : undefined,
        'data-target': props.target,
        'data-rel': props.rel,
        'data-download': props.download,
        'data-ping': props.ping,
        'data-referrer-policy': props.referrerPolicy
    };
}
function $4068a0fae83b6d84$export$51437d503373d223(props) {
    return {
        'data-href': props.href,
        'data-target': props.target,
        'data-rel': props.rel,
        'data-download': props.download,
        'data-ping': props.ping,
        'data-referrer-policy': props.referrerPolicy
    };
}
function $4068a0fae83b6d84$export$7e924b3091a3bd18(props) {
    let router = $4068a0fae83b6d84$export$9a302a45f65d0572();
    var _props_href;
    const href = router.useHref((_props_href = props === null || props === void 0 ? void 0 : props.href) !== null && _props_href !== void 0 ? _props_href : '');
    return {
        href: (props === null || props === void 0 ? void 0 : props.href) ? href : undefined,
        target: props === null || props === void 0 ? void 0 : props.target,
        rel: props === null || props === void 0 ? void 0 : props.rel,
        download: props === null || props === void 0 ? void 0 : props.download,
        ping: props === null || props === void 0 ? void 0 : props.ping,
        referrerPolicy: props === null || props === void 0 ? void 0 : props.referrerPolicy
    };
}


//# sourceMappingURL=openLink.main.js.map
