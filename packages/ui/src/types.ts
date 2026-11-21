// The prop vocabularies every Weft component shares (see the plan's Global
// Constraints): one meaning for `size`, `tone` and `variant` everywhere.

/** `sm`, `md` or `lg`, from the theme's control sizes. */
export type Size = 'sm' | 'md' | 'lg'

/** What a colour says: `neutral`, `primary`, `danger`, `warning` or `success`. */
export type Tone = 'neutral' | 'primary' | 'danger' | 'warning' | 'success'

/** How strongly a button or badge is drawn: filled, tinted, outlined or bare. */
export type Variant = 'solid' | 'soft' | 'outline' | 'ghost'
