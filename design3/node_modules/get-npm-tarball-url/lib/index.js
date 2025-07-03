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

// src/index.ts
var src_exports = {};
__export(src_exports, {
  default: () => src_default
});
module.exports = __toCommonJS(src_exports);
function src_default(pkgName, pkgVersion, opts) {
  let registry;
  if (opts == null ? void 0 : opts.registry) {
    registry = opts.registry.endsWith("/") ? opts.registry : `${opts.registry}/`;
  } else {
    registry = "https://registry.npmjs.org/";
  }
  const scopelessName = getScopelessName(pkgName);
  return `${registry}${pkgName}/-/${scopelessName}-${removeBuildMetadataFromVersion(pkgVersion)}.tgz`;
}
function removeBuildMetadataFromVersion(version) {
  const plusPos = version.indexOf("+");
  if (plusPos === -1)
    return version;
  return version.substring(0, plusPos);
}
function getScopelessName(name) {
  if (name[0] !== "@") {
    return name;
  }
  return name.split("/")[1];
}
//# sourceMappingURL=index.js.map