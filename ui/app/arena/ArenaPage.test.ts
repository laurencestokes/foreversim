import { SimSettingCategories } from '@sim/constants/sim_settings';
import { PlayerSpecs } from '@sim/player/specs';
import { tryParseUrlLocation } from '@sim/state/sim_links';
import { describe, expect, it } from 'vitest';

import { bestPerTree, talentLink } from './ArenaPage';

describe('arena talent links', () => {
	it('open the spec page with only the talents', () => {
		const link = new URL(talentLink(PlayerSpecs.DpsWarrior, '30305001302-05050005525010051'), 'https://example.com');
		expect(link.pathname).toBe(PlayerSpecs.DpsWarrior.simLink);
		const parsed = tryParseUrlLocation(link);
		expect(parsed?.categories).toEqual([SimSettingCategories.Talents]);
		expect(parsed?.settings.player?.talentsString).toBe('30305001302-05050005525010051');
	});
});

describe('arena default view', () => {
	it('keeps the best build in each tree of a spec', () => {
		const row = (talents: string, dps: number) => ({ spec: 'warrior', talents, dps }) as Parameters<typeof bestPerTree>[0][number];
		const shown = bestPerTree([row('34300003-550500005152310051', 715), row('3-550500005152310051', 700), row('55050103201-0505', 650)]);
		expect(shown.map(b => b.dps)).toEqual([715, 650]);
	});
});
