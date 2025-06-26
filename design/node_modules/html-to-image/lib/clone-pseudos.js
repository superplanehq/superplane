"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.clonePseudoElements = void 0;
var util_1 = require("./util");
function formatCSSText(style) {
    var content = style.getPropertyValue('content');
    return "".concat(style.cssText, " content: '").concat(content.replace(/'|"/g, ''), "';");
}
function formatCSSProperties(style, options) {
    return (0, util_1.getStyleProperties)(options)
        .map(function (name) {
        var value = style.getPropertyValue(name);
        var priority = style.getPropertyPriority(name);
        return "".concat(name, ": ").concat(value).concat(priority ? ' !important' : '', ";");
    })
        .join(' ');
}
function getPseudoElementStyle(className, pseudo, style, options) {
    var selector = ".".concat(className, ":").concat(pseudo);
    var cssText = style.cssText
        ? formatCSSText(style)
        : formatCSSProperties(style, options);
    return document.createTextNode("".concat(selector, "{").concat(cssText, "}"));
}
function clonePseudoElement(nativeNode, clonedNode, pseudo, options) {
    var style = window.getComputedStyle(nativeNode, pseudo);
    var content = style.getPropertyValue('content');
    if (content === '' || content === 'none') {
        return;
    }
    var className = (0, util_1.uuid)();
    try {
        clonedNode.className = "".concat(clonedNode.className, " ").concat(className);
    }
    catch (err) {
        return;
    }
    var styleElement = document.createElement('style');
    styleElement.appendChild(getPseudoElementStyle(className, pseudo, style, options));
    clonedNode.appendChild(styleElement);
}
function clonePseudoElements(nativeNode, clonedNode, options) {
    clonePseudoElement(nativeNode, clonedNode, ':before', options);
    clonePseudoElement(nativeNode, clonedNode, ':after', options);
}
exports.clonePseudoElements = clonePseudoElements;
//# sourceMappingURL=clone-pseudos.js.map