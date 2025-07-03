
function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "getOwnerDocument", () => $aaa611146751592e$export$b204af158042fbac);
$parcel$export(module.exports, "getOwnerWindow", () => $aaa611146751592e$export$f21a1ffae260145a);
$parcel$export(module.exports, "isShadowRoot", () => $aaa611146751592e$export$af51f0f06c0f328a);
const $aaa611146751592e$export$b204af158042fbac = (el)=>{
    var _el_ownerDocument;
    return (_el_ownerDocument = el === null || el === void 0 ? void 0 : el.ownerDocument) !== null && _el_ownerDocument !== void 0 ? _el_ownerDocument : document;
};
const $aaa611146751592e$export$f21a1ffae260145a = (el)=>{
    if (el && 'window' in el && el.window === el) return el;
    const doc = $aaa611146751592e$export$b204af158042fbac(el);
    return doc.defaultView || window;
};
/**
 * Type guard that checks if a value is a Node. Verifies the presence and type of the nodeType property.
 */ function $aaa611146751592e$var$isNode(value) {
    return value !== null && typeof value === 'object' && 'nodeType' in value && typeof value.nodeType === 'number';
}
function $aaa611146751592e$export$af51f0f06c0f328a(node) {
    return $aaa611146751592e$var$isNode(node) && node.nodeType === Node.DOCUMENT_FRAGMENT_NODE && 'host' in node;
}


//# sourceMappingURL=domHelpers.main.js.map
