define(['./_tagTester', './_methodFingerprint', './_stringTagBug'], function (_tagTester, _methodFingerprint, _stringTagBug) {

	var isMap = _stringTagBug.isIE11 ? _methodFingerprint.ie11fingerprint(_methodFingerprint.mapMethods) : _tagTester('Map');

	return isMap;

});
