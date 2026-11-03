export type LaunchPreview = {
	launchDate: string;
	notesUrl?: string;
	upstreamCommit: string;
	drafts: Array<{ pullRequest: number; commit: string }>;
};
