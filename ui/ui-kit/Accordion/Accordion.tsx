import { Accordion as BaseAccordion } from '@base-ui/react/accordion';
import type { ClassValue } from 'clsx';
import clsx from 'clsx';
import { createContext, type ReactNode } from 'react';

/** `divided` rules the items apart; `boxed` draws each item as its own bordered box, with a gap between them. */
export type AccordionVariant = 'divided' | 'boxed';

export const AccordionVariantContext = createContext<AccordionVariant>('divided');

/** `compact` sets the item titles a step smaller and lighter, for an accordion that sits under a heading of its own. */
export type AccordionDensity = 'default' | 'compact';

export interface AccordionProps {
	value: string[];
	onValueChange: (value: string[]) => void;
	variant?: AccordionVariant;
	density?: AccordionDensity;
	className?: ClassValue;
	testId?: string;
	children?: ReactNode;
}

export const Accordion = ({ value, onValueChange, variant = 'divided', density = 'default', className, testId = 'accordion', children }: AccordionProps) => (
	<BaseAccordion.Root
		className={clsx('ui-accordion', className)}
		multiple
		value={value}
		onValueChange={onValueChange}
		data-variant={variant}
		data-density={density}
		data-testid={testId}>
		<AccordionVariantContext value={variant}>{children}</AccordionVariantContext>
	</BaseAccordion.Root>
);
