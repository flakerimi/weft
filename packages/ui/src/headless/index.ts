// Weft's headless layer: behaviour without markup or styles, shared by every
// component. Each file's header documents its contract.
export { overlayStack, type OverlayEntry } from './overlayStack.ts'
export { useControllable } from './useControllable.ts'
export { useDismiss, type DismissOptions, type DismissReason } from './useDismiss.ts'
export { arrow, hide, useFloating, type Floating, type FloatingOptions, type Middleware, type MiddlewareData, type Placement } from './useFloating.ts'
export { useFocusTrap, type FocusTrapOptions } from './useFocusTrap.ts'
export { ID_SCOPE, useId, type IdScope } from './useId.ts'
export { useRovingFocus, type RovingFocus, type RovingFocusOptions } from './useRovingFocus.ts'
export { useScrollLock } from './useScrollLock.ts'
export { useInert } from './useInert.ts'
export { createSwipe, type Swipe, type SwipeDirection } from './swipe.ts'
export { useTypeahead, type Typeahead } from './useTypeahead.ts'
