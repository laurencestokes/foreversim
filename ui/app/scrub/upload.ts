// Sending beta client files is switched off on this site: nothing a visitor drops on the scrub page
// leaves their browser. (Elliot Wood's site sends them to his own collector; this copy has none.)
// To turn it back on, point UPLOAD_URL at a collector of our own and restore the fetch below from
// history.

export const UPLOAD_URL = '';

export type UploadKind = 'dbcache' | 'damagemeter' | 'screenshot';

export type UploadResult = { ok: true; receipt: string } | { ok: false; error: string };

export async function upload(_kind: UploadKind, _bytes: Uint8Array, _note: string): Promise<UploadResult> {
	return { ok: false, error: 'This site does not collect files. Nothing was sent.' };
}
