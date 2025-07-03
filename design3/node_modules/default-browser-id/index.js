import os from 'os';
import {promises as fs} from 'fs';
import bplist from 'bplist-parser';
import untildify from 'untildify';

const macOsVersion = Number(os.release().split('.')[0]);
const filePath = untildify(macOsVersion >= 14 ? '~/Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist' : '~/Library/Preferences/com.apple.LaunchServices.plist');

export default async function defaultBrowserId() {
	if (process.platform !== 'darwin') {
		throw new Error('macOS only');
	}

	let bundleId = 'com.apple.Safari';

	let buffer;
	try {
		buffer = await fs.readFile(filePath);
	} catch (error) {
		if (error.code === 'ENOENT') {
			return bundleId;
		}

		throw error;
	}

	const data = bplist.parseBuffer(buffer);
	const handlers = data && data[0].LSHandlers;

	if (!handlers || handlers.length === 0) {
		return bundleId;
	}

	for (const handler of handlers) {
		if (handler.LSHandlerURLScheme === 'http' && handler.LSHandlerRoleAll) {
			bundleId = handler.LSHandlerRoleAll;
			break;
		}
	}

	return bundleId;
}
