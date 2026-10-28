import '../shared/page_boot';

import i18n from '@i18n/config';
import { PortalContainerContext } from '@ui-kit/hooks/usePortalContainer';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { BuildArenaPage } from './build_arena/BuildArenaPage';

document.title = `${i18n.t('title', { ns: 'build_arena' })} - WoWSims Forever`;

const rootElem = document.getElementById('root');
if (!rootElem) throw new Error('No #root element on the build arena page.');

createRoot(rootElem).render(
	<StrictMode>
		<PortalContainerContext value={rootElem}>
			<BuildArenaPage />
		</PortalContainerContext>
	</StrictMode>,
);
