"use strict";
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// src/dom-event.ts
var dom_event_exports = {};
__export(dom_event_exports, {
  extractEventHiddenProperties: () => extractEventHiddenProperties
});
module.exports = __toCommonJS(dom_event_exports);
var eventProperties = [
  "bubbles",
  "cancelBubble",
  "cancelable",
  "composed",
  "currentTarget",
  "defaultPrevented",
  "eventPhase",
  "isTrusted",
  "returnValue",
  "srcElement",
  "target",
  "timeStamp",
  "type"
];
var customEventSpecificProperties = ["detail"];
function extractEventHiddenProperties(event) {
  const rebuildEvent = eventProperties.filter((value) => event[value] !== void 0).reduce((acc, value) => {
    return { ...acc, [value]: event[value] };
  }, {});
  if (event instanceof CustomEvent) {
    customEventSpecificProperties.filter((value) => event[value] !== void 0).forEach((value) => {
      rebuildEvent[value] = event[value];
    });
  }
  return rebuildEvent;
}
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  extractEventHiddenProperties
});
