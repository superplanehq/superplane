define(['./_tagTester', './_methodFingerprint', './_stringTagBug'], function (_tagTester, _methodFingerprint, _stringTagBug) {

	var isSet = _stringTagBug.isIE11 ? _methodFingerprint.ie11fingerprint(_methodFingerprint.setMethods) : _tagTester('Set');

	return isSet;

});
