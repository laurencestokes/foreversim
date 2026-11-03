export const launchPreviewFixture = {
	launchDate: '2026-11-04',
	upstreamCommit: 'a'.repeat(40),
	drafts: [824, 823, 822, 821].map((pullRequest, index) => ({ pullRequest, commit: String(index + 1).repeat(40) })),
};
