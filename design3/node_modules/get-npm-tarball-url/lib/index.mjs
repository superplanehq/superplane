// src/index.ts
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
export {
  src_default as default
};
//# sourceMappingURL=index.mjs.map