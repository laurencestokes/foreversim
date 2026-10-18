import '../shared/page_boot';

import { PortalContainerContext } from '@ui-kit/hooks/usePortalContainer';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { RaceArenaPage } from './race_arena/RaceArenaPage';

document.title = 'Race tier list - WoWSims Forever';

const rootElem = document.getElementById('root');
if (!rootElem) throw new Error('No #root element on the page; ui/index.html should provide it.');

createRoot(rootElem).render(
	<StrictMode>
		<PortalContainerContext value={rootElem}>
			<RaceArenaPage />
		</PortalContainerContext>
	</StrictMode>,
);
