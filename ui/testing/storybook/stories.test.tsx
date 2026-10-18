import { readFile } from 'node:fs/promises';

import { Database } from '@sim/proto/database';
import { composeStories, type Meta, setProjectAnnotations, type StoryObj } from '@storybook/react-vite';
import { beforeAll, describe, expect, it, vi } from 'vitest';

import * as preview from '../../../.storybook/preview';

const annotations = setProjectAnnotations([preview]);
beforeAll(annotations.beforeAll);
beforeAll(async () => {
	// Stories resolve real item icons, but the test harness has no server to serve the database.
	const json = await readFile('assets/database/db.json', 'utf8');
	const fetchDatabase = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(json));
	try {
		await Database.get();
	} finally {
		fetchDatabase.mockRestore();
	}
});

type StoriesModule = { default: Meta } & Record<string, StoryObj>;

const modules = import.meta.glob<StoriesModule>('../../**/*.stories.tsx', { eager: true });

describe.each(Object.entries(modules))('%s', (_file, mod) => {
	const stories = Object.entries(composeStories(mod));
	it.each(stories)('%s mounts', async (_name, story) => {
		if (story.parameters?.smoke === false) {
			expect(story.parameters.smokeReason).toMatch(/\S/);
			return;
		}
		const canvasElement = document.body.appendChild(document.createElement('div'));
		try {
			await story.run({ canvasElement });
			expect(canvasElement.childElementCount).toBeGreaterThan(0);
		} finally {
			canvasElement.remove();
		}
	});
});
