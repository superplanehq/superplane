
function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "isMac", () => $9e20cff0af27e8cc$export$9ac100e40613ea10);
$parcel$export(module.exports, "isIPhone", () => $9e20cff0af27e8cc$export$186c6964ca17d99);
$parcel$export(module.exports, "isIPad", () => $9e20cff0af27e8cc$export$7bef049ce92e4224);
$parcel$export(module.exports, "isIOS", () => $9e20cff0af27e8cc$export$fedb369cb70207f1);
$parcel$export(module.exports, "isAppleDevice", () => $9e20cff0af27e8cc$export$e1865c3bedcd822b);
$parcel$export(module.exports, "isWebKit", () => $9e20cff0af27e8cc$export$78551043582a6a98);
$parcel$export(module.exports, "isChrome", () => $9e20cff0af27e8cc$export$6446a186d09e379e);
$parcel$export(module.exports, "isAndroid", () => $9e20cff0af27e8cc$export$a11b0059900ceec8);
$parcel$export(module.exports, "isFirefox", () => $9e20cff0af27e8cc$export$b7d78993b74f766d);
/*
 * Copyright 2020 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ function $9e20cff0af27e8cc$var$testUserAgent(re) {
    var _window_navigator_userAgentData;
    if (typeof window === 'undefined' || window.navigator == null) return false;
    return ((_window_navigator_userAgentData = window.navigator['userAgentData']) === null || _window_navigator_userAgentData === void 0 ? void 0 : _window_navigator_userAgentData.brands.some((brand)=>re.test(brand.brand))) || re.test(window.navigator.userAgent);
}
function $9e20cff0af27e8cc$var$testPlatform(re) {
    var _window_navigator_userAgentData;
    return typeof window !== 'undefined' && window.navigator != null ? re.test(((_window_navigator_userAgentData = window.navigator['userAgentData']) === null || _window_navigator_userAgentData === void 0 ? void 0 : _window_navigator_userAgentData.platform) || window.navigator.platform) : false;
}
function $9e20cff0af27e8cc$var$cached(fn) {
    if (process.env.NODE_ENV === 'test') return fn;
    let res = null;
    return ()=>{
        if (res == null) res = fn();
        return res;
    };
}
const $9e20cff0af27e8cc$export$9ac100e40613ea10 = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testPlatform(/^Mac/i);
});
const $9e20cff0af27e8cc$export$186c6964ca17d99 = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testPlatform(/^iPhone/i);
});
const $9e20cff0af27e8cc$export$7bef049ce92e4224 = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testPlatform(/^iPad/i) || // iPadOS 13 lies and says it's a Mac, but we can distinguish by detecting touch support.
    $9e20cff0af27e8cc$export$9ac100e40613ea10() && navigator.maxTouchPoints > 1;
});
const $9e20cff0af27e8cc$export$fedb369cb70207f1 = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$export$186c6964ca17d99() || $9e20cff0af27e8cc$export$7bef049ce92e4224();
});
const $9e20cff0af27e8cc$export$e1865c3bedcd822b = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$export$9ac100e40613ea10() || $9e20cff0af27e8cc$export$fedb369cb70207f1();
});
const $9e20cff0af27e8cc$export$78551043582a6a98 = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testUserAgent(/AppleWebKit/i) && !$9e20cff0af27e8cc$export$6446a186d09e379e();
});
const $9e20cff0af27e8cc$export$6446a186d09e379e = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testUserAgent(/Chrome/i);
});
const $9e20cff0af27e8cc$export$a11b0059900ceec8 = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testUserAgent(/Android/i);
});
const $9e20cff0af27e8cc$export$b7d78993b74f766d = $9e20cff0af27e8cc$var$cached(function() {
    return $9e20cff0af27e8cc$var$testUserAgent(/Firefox/i);
});


//# sourceMappingURL=platform.main.js.map
