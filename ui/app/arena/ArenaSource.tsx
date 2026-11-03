import i18n from '@i18n/config';
import { Button } from '@ui-kit/Button';

export type ArenaSource = {
	id: string;
	commit: string;
	publishedAt?: string;
	computedAt?: string;
	computedWindow?: { start: string; end: string };
	reused: boolean;
	note: string;
};

export const sourceDate = (source: ArenaSource) =>
	source.computedAt ??
	(source.computedWindow ? i18n.t('computedWindow', { ns: 'build_arena', ...source.computedWindow }) : i18n.t('dateUnrecorded', { ns: 'build_arena' }));

export const MeasurementSource = ({ source }: { source: ArenaSource }) => (
	<span className="text-xs text-white/60">
		{i18n.t(source.reused ? 'measurementReused' : 'measurementFresh', { ns: 'build_arena', date: sourceDate(source) })} ·{' '}
		<Button variant="link" href={`https://github.com/laurencestokes/foreversim/commit/${source.commit}`} target="_blank">
			<code>{source.commit.slice(0, 7)}</code>
		</Button>
	</span>
);
