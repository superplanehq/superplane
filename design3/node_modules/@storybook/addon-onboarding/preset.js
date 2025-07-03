function managerEntries(entry = []) {
  return [...entry, require.resolve("./dist/manager.mjs")];
}

module.exports = {
  managerEntries,
  ...require("./dist/preset")
}
