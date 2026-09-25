// ForeverSim sends no analytics. The trackers stay as no-ops so the call sites shared with
// upstream need no changes.
export const trackPageView = (_title: string, _slug: string) => {};

export type TrackEventProps = {
	action: 'settings' | 'sim' | 'click';
	category: string;
	label?: string;
	value?: number | string | boolean;
	additionalData?: Record<string, string | number>;
};

export const trackEvent = (_event: TrackEventProps) => {};
