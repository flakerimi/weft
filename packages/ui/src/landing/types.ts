import type { AccordionItem } from '../disclosure/accordion.ts'
import type { ActionItem } from '../feedback/actions.ts'

/** One capability or benefit in a FeatureGrid. */
export interface FeatureItem {
  /** The feature's heading. */
  title: string
  /** A sentence or two explaining the feature. */
  description?: string
  /** A Lucide icon drawn in a tinted disc above the title. */
  icon?: string
  /** An address: makes the feature's link point there. */
  to?: string
  /** Text for the item's standalone link when `to` is set; without it, the title itself links to `to`. */
  linkLabel?: string
  /** The name of the FeatureGrid's slot that renders custom content for this item. */
  slot?: string
}

/** A billing period choice in a Pricing table's ToggleGroup (`monthly`, `yearly`). */
export interface PricingCycle {
  /** Its id in `v-model:cycle`. */
  value: string
  /** The button's label ("Monthly", "Yearly"). */
  label: string
  /** An optional badge beside the label ("Save 20%"). */
  badge?: string
}

/** One plan in a Pricing comparison. */
export interface PricingPlan {
  /** Identifies the plan; default its title. */
  id?: string
  /** The plan's name ("Starter", "Pro"). */
  title: string
  /** Who the plan is for. */
  description?: string
  /** The price as shown ("$29", "Free"), or by cycle value (`{ monthly: '$29', yearly: '$24' }`). A map lacking the selected cycle shows an empty price area and warns once in development. */
  price: string | Record<string, string>
  /** After the price ("/ month"), or by cycle value (a map lacking the selected cycle shows nothing and warns once in development). */
  period?: string | Record<string, string>
  /** A line under the price ("Billed $288 yearly"), or by cycle value (a missing cycle shows no note). */
  billingNote?: string | Record<string, string>
  /** A badge on the plan's card ("Most popular"). */
  badge?: string
  /** Emphasise this plan with a strong border in the section's tone and a solid primary button. */
  highlight?: boolean
  /** What the plan includes, listed with a check icon. */
  features?: string[]
  /** The plan's call-to-action button. */
  action?: ActionItem
  /** The name of the Pricing's slot that renders custom body content for this plan. */
  slot?: string
}

/** One customer quote in Testimonials. */
export interface TestimonialItem {
  /** The words quoted. */
  quote: string
  /** Who said it ("Avery Quinn"): names the author and gives the avatar's fallback initials. */
  author: string
  /** Their role, company or context ("CTO at Northline"). */
  role?: string
  /** The avatar image's address. */
  avatar?: string
  /** An address: the author's name links there. */
  to?: string
}

/** One question and answer in an Faq: the same shape as AccordionItem (`{ label, content, value, icon, trailingIcon, disabled, slot }`). */
export type FaqItem = AccordionItem
