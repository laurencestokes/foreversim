import i18n from '@i18n/config';
import { Button } from '@ui-kit/Button';

import type { LaunchPreview } from './launchPreview';

export const LaunchPreviewNotice = ({ preview, commit }: { preview: LaunchPreview; commit: string }) => {
	const t = (key: string, values?: Record<string, unknown>) => i18n.t(`build_arena:${key}`, values);
	const classHotfix = preview.drafts.some(source => source.pullRequest === 840);
	const date = new Intl.DateTimeFormat(i18n.language, { dateStyle: 'long', timeZone: 'UTC' }).format(new Date(`${preview.launchDate}T00:00:00Z`));
	return (
		<aside className="flex flex-col gap-2 border border-orange bg-orange/10 p-4" aria-label={t('launchPreviewTitle', { date })}>
			<h2 className="m-0 text-xl text-orange">{t('launchPreviewTitle', { date })}</h2>
			<p className="m-0">{t(classHotfix ? 'hotfixPreviewSummary' : 'launchPreviewSummary')}</p>
			{preview.notesUrl && (
				<p className="m-0">
					{t(classHotfix ? 'hotfixPreviewDetails' : 'launchPreviewGaps')}{' '}
					<Button variant="link" href={`https://github.com/laurencestokes/foreversim/blob/${commit}/docs/november-2-coverage.md`} target="_blank">
						{t('launchPreviewCoverageLink')}
					</Button>
				</p>
			)}
			{preview.notesUrl && <p className="m-0 text-sm text-white/70">{t(classHotfix ? 'hotfixPreviewLimits' : 'launchPreviewShamanSlot')}</p>}
			<p className="m-0">{t('launchPreviewStatus')}</p>
			<nav className="flex flex-wrap items-baseline gap-x-4 gap-y-1 text-sm" aria-label={t('launchPreviewSources')}>
				{preview.notesUrl && (
					<Button variant="link" href={preview.notesUrl} target="_blank">
						{t('launchPreviewNotes')}
					</Button>
				)}
				<Button variant="link" href={`https://github.com/laurencestokes/foreversim/commit/${commit}`} target="_blank">
					{t('launchPreviewEngine')} {commit.slice(0, 7)}
				</Button>
				<Button variant="link" href={`https://github.com/wowsims/forever/commit/${preview.upstreamCommit}`} target="_blank">
					{t('launchPreviewBase')} {preview.upstreamCommit.slice(0, 7)}
				</Button>
				{preview.drafts.map(source => (
					<span key={source.pullRequest} className="inline-flex items-baseline gap-1">
						<Button variant="link" href={`https://github.com/wowsims/forever/pull/${source.pullRequest}`} target="_blank">
							{t('launchPreviewDraft', { number: source.pullRequest })}
						</Button>
						<Button variant="link" href={`https://github.com/wowsims/forever/commit/${source.commit}`} target="_blank">
							{source.commit.slice(0, 7)}
						</Button>
					</span>
				))}
			</nav>
		</aside>
	);
};
