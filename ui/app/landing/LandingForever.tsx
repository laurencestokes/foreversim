import { Icon, type IconName } from '@ui-kit/Icon';
import { SimLinkContent } from '@ui-kit/SimLinkContent';
import type { ReactNode } from 'react';

import { SITE_BASE, SITE_REPO_URL } from '../ProductPage/site';

// What the Forever landing page says above the class menus: what this sim is, who it owes, how
// to help, and the links to its product pages. Hand-written English, as it was before the port.

const PRODUCT_LINKS: Array<{ href: string; title: string; status: string }> = [
	{ href: 'bis/', title: 'Best in Slot', status: 'Launch - Alpha' },
	{ href: 'dps_rankings/', title: 'Damage comparison', status: 'Every community build, one raid' },
	{ href: 'stat_weights/', title: 'Stat Weights', status: 'Launch - Alpha' },
	{ href: 'changelog/', title: 'What changed for Forever', status: 'Changelog and sources' },
	{ href: 'arena/', title: 'The build arena', status: 'Every build, ranked' },
	{ href: 'evidence/', title: 'Where every number came from', status: '996 abilities, one reason each' },
];

const CTA = 'inline-flex items-center gap-2 rounded-sm border border-brand px-4 py-2 font-bold no-underline';
const CTA_LOUD = `${CTA} bg-brand text-black hover:bg-brand/80`;
const CTA_QUIET = `${CTA} text-brand hover:bg-brand/20`;

const Cta = ({ href, icon, quiet, children }: { href: string; icon: IconName; quiet?: boolean; children: ReactNode }) => (
	<a className={quiet ? CTA_QUIET : CTA_LOUD} href={`${SITE_BASE}${href}`}>
		<Icon name={icon} />
		<span>{children}</span>
	</a>
);

const Panel = ({ summary, children }: { summary: string; children: ReactNode }) => (
	<details className="border border-surface-border bg-black/50 px-4 py-3">
		<summary className="cursor-pointer text-lg font-bold">{summary}</summary>
		<div className="flex flex-col gap-3 pt-3">{children}</div>
	</details>
);

export const LandingForever = () => (
	<div className="flex w-3/4 flex-col gap-4 max-lg:w-full" data-testid="landing-forever">
		<p id="description" className="m-0 text-fluid-xl">
			An unofficial sim for World of Warcraft®: Forever, the Classic+ relaunch announced at BlizzCon 2026. It carries Forever&apos;s talent trees, races
			and ruleset through every spec.
		</p>
		<p className="m-0 opacity-80">
			<strong>Not affiliated with Blizzard or the WoWSims team.</strong> Every number is provisional &mdash; useful for catching the sim doing something
			obviously wrong, not as a statement about Forever. Problems and evidence are welcome as{' '}
			<a href={`${SITE_REPO_URL}/issues`} target="_blank" rel="noreferrer">
				GitHub issues
			</a>
			.
		</p>
		<p className="m-0 opacity-80" data-testid="wowsims-credit">
			Built on the open-source{' '}
			<a href="https://github.com/wowsims" target="_blank" rel="noreferrer">
				WoWSims
			</a>{' '}
			simulators and{' '}
			<a href="https://github.com/ElliotWood/Forever" target="_blank" rel="noreferrer">
				Elliot Wood&apos;s Forever sim
			</a>
			, used under their{' '}
			<a href={`${SITE_REPO_URL}/blob/master/LICENSE`} target="_blank" rel="noreferrer">
				MIT licence
			</a>
			. The engine and most of the Forever modelling are their work.
		</p>
		<p className="m-0 flex flex-wrap gap-3">
			<Cta href="arena/" icon="trophy">
				The build arena
			</Cta>
			<Cta href="evidence/" icon="clipboard-check" quiet>
				Where every number came from
			</Cta>
			<Cta href="evidence/#most-wanted" icon="hand-holding-heart" quiet>
				How you can help
			</Cta>
		</p>
		<Panel summary="Where the numbers come from">
			<p className="m-0">
				Since the beta client was datamined on 17 September the numbers come from the client&apos;s own data tables rather than BlizzCon tooltips: build{' '}
				<code>1.60.1.69913</code> on wago.tools, read against Classic Era and diffed spell by spell. Talent values come from the client&apos;s rank
				curves, coefficients from <code>SpellEffect.EffectBonusCoefficient</code>, and each rank&apos;s damage is scaled to level 60 by the
				client&apos;s own per-level points.
			</p>
			<p className="m-0">Most of it is settled from data, a little of it has been seen happen, and it is worth being plain about which is which:</p>
			<ul className="m-0 flex flex-col gap-2 pl-5">
				<li>
					<strong>The client&apos;s tables, diffed against Classic Era.</strong> 656 abilities carry the client&apos;s numbers and 297 are confirmed
					unchanged from Classic.
				</li>
				<li>
					<strong>Hotfixes, which the static data does not carry.</strong> The live client&apos;s own cache is read instead. Today it holds 11,800
					changed rows, all item data, with a 4 byte stub for spells.
				</li>
				<li>
					<strong>Wording, not just values.</strong> A rank curve says what a talent&apos;s numbers are, not what they apply to, which is how Improved
					Seals passed a value check while scaling half of what it should. 231 talents read differently in Forever.
				</li>
				<li>
					<strong>Twelve abilities have now been watched happen on a running server</strong> &mdash; through the client&apos;s own damage meter, on
					the beta &mdash; and every one landed where the client&apos;s tables said it would. Twelve out of 996, so the honest reading is that the
					method works, not that the sim is verified. The other 984 are internally consistent and externally unconfirmed.
				</li>
			</ul>
			<p className="m-0">
				<a href={`${SITE_BASE}evidence/`}>Every ability, and what is known about its numbers</a> &mdash; the whole list, filterable, with the reason
				attached to each row.
			</p>
		</Panel>
		<Panel summary="What it still cannot do">
			<p className="m-0">Every one of these can move a number, and some can move it a long way:</p>
			<ul className="m-0 flex flex-col gap-2 pl-5">
				<li>
					43 abilities still carry a number the client does not settle &mdash; 25 of them hunter. Each is named on the{' '}
					<a href={`${SITE_BASE}evidence/`}>evidence page</a>, in the spell manifest, and in the{' '}
					<a href={`${SITE_REPO_URL}/blob/master/docs/forever_beta_checklist.md`} target="_blank" rel="noreferrer">
						beta re-verification checklist
					</a>
					.
				</li>
				<li>
					<strong>Downranking is unresolved, and it is the big one.</strong> The client carries the full coefficient on low ranks where Classic Era
					carried a reduced one. Read as written, a rank 4 Lightning Bolt does most of a rank 10 for a quarter of the mana. Whether Forever removed
					the penalty changes every caster here.
				</li>
				<li>
					The client stores what an ability does, not how the server runs it. Proc chances often read as unset, and rules like whether melee-table
					Holy damage partially resists are in no table.
				</li>
				<li>
					No healing sim runs, tank specs are measured on damage alone, and rotations are hand-written priority lists that nothing has re-tuned around
					the cooldowns Forever changed.
				</li>
			</ul>
		</Panel>
	</div>
);

// The product pages, as a row of the same cells the class menus use.
export const LandingProductLinks = () => (
	<div className="flex flex-wrap" data-testid="product-links">
		{PRODUCT_LINKS.map(link => (
			<div key={link.href} className="ui-landing-sim-link-dropdown">
				<a href={`${SITE_BASE}${link.href}`} className="ui-landing-sim-link-cell text-white" data-testid="product-link">
					<SimLinkContent iconPath={`${SITE_BASE}assets/img/WoW-Simulator-Icon.png`} title={link.title} status={link.status} />
				</a>
			</div>
		))}
	</div>
);
