define(['./_escapeMap', './invert'], function (_escapeMap, invert) {

	// Internal list of HTML entities for unescaping.
	var unescapeMap = invert(_escapeMap);

	return unescapeMap;

});
