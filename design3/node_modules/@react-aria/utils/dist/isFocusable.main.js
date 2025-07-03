
function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "isFocusable", () => $506b33fd893eab7d$export$4c063cf1350e6fed);
$parcel$export(module.exports, "isTabbable", () => $506b33fd893eab7d$export$bebd5a1431fec25d);
const $506b33fd893eab7d$var$focusableElements = [
    'input:not([disabled]):not([type=hidden])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    'button:not([disabled])',
    'a[href]',
    'area[href]',
    'summary',
    'iframe',
    'object',
    'embed',
    'audio[controls]',
    'video[controls]',
    '[contenteditable]:not([contenteditable^="false"])'
];
const $506b33fd893eab7d$var$FOCUSABLE_ELEMENT_SELECTOR = $506b33fd893eab7d$var$focusableElements.join(':not([hidden]),') + ',[tabindex]:not([disabled]):not([hidden])';
$506b33fd893eab7d$var$focusableElements.push('[tabindex]:not([tabindex="-1"]):not([disabled])');
const $506b33fd893eab7d$var$TABBABLE_ELEMENT_SELECTOR = $506b33fd893eab7d$var$focusableElements.join(':not([hidden]):not([tabindex="-1"]),');
function $506b33fd893eab7d$export$4c063cf1350e6fed(element) {
    return element.matches($506b33fd893eab7d$var$FOCUSABLE_ELEMENT_SELECTOR);
}
function $506b33fd893eab7d$export$bebd5a1431fec25d(element) {
    return element.matches($506b33fd893eab7d$var$TABBABLE_ELEMENT_SELECTOR);
}


//# sourceMappingURL=isFocusable.main.js.map
