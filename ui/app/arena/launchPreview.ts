export type LaunchPreview = {
	launchDate: string;
	upstreamCommit: string;
	drafts: Array<{ pullRequest: number; commit: string }>;
};
