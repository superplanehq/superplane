module.exports = {
  testMatch: ['**/*.test.ts'],
  // transform everything including node_modules
  transformIgnorePatterns: [],
  // deal with missing main fields
  moduleNameMapper: {
    'estree-walker': 'estree-walker/src',
    'is-reference': 'is-reference/src',
  },
};
