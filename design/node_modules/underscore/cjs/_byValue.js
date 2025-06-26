// Internal wrapper to enable match-by-value mode in `linearSearch`.
function byValue(value) {
  if (!(this instanceof byValue)) return new byValue(value);
  this.value = value;
}

module.exports = byValue;
