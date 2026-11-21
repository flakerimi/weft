/**
 * useModal(props, emit, { prefix, wrap, panel }): what Dialog, Sheet and
 * Drawer share, the WAI-ARIA APG Dialog (Modal) pattern.
 *
 * - `open` follows `v-model:open` (or `defaultOpen`); the trigger, the
 *   first element of the default slot inside `wrap`, opens it and carries
 *   `aria-haspopup="dialog"`, `aria-expanded` and `aria-controls`.
 * - While open, focus is trapped in `panel` and the page does not scroll.
 *   Focus starts on the first tabbable element that is not the close
 *   button (the close button when it is the only one, else the panel).
 * - Escape closes it, and so does a press on its backdrop (`onBackdrop`),
 *   both only when `dismissible`. A press anywhere else (a toast above it)
 *   is not taken. Focus returns to the trigger (`returnTo`), or to the
 *   element focused when it opened if it has none, after a press on the
 *   backdrop too: the backdrop is the modal's own, so the overlay stack's
 *   rule of no focus return after a press outside does not apply.
 * - While open, the page behind is inert (useInert), the toaster aside.
 * - Closing plays the leave transition (`present` stays true while
 *   `closing`); the stack, the trap, the lock and the inert page let go at
 *   once, so focus is home before the panel has gone.
 * - `context` carries the direction, scheme, input and surface of where it
 *   opened into the overlay root; `ids` names the panel, title and
 *   description for `aria-controls`, `aria-labelledby`, `aria-describedby`.
 */
import { computed, useTemplateRef, watch, type Ref } from 'vue'
import { tabbables } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { useFocusTrap } from '../headless/useFocusTrap.ts'
import { useId } from '../headless/useId.ts'
import { useInert } from '../headless/useInert.ts'
import { useScrollLock } from '../headless/useScrollLock.ts'
import { useMounted, useOverlayContext, usePresence, useTrigger } from './overlay.ts'

export interface ModalProps {
  open?: boolean
  defaultOpen?: boolean
  dismissible?: boolean
}

export const CLOSE_CLASS = 'weft-overlay__close'

export function useModal(
  props: ModalProps,
  emit: (event: 'update:open', open: boolean) => void,
  opts: { prefix: string },
) {
  const open = useControllable(props, 'open', emit, () => props.defaultOpen ?? false) as Ref<boolean>
  const mounted = useMounted()
  const id = useId(opts.prefix)
  const ids = { panel: id, title: `${id}-title`, description: `${id}-description` }
  const wrap = useTemplateRef<HTMLElement>('wrap')
  const panel = useTemplateRef<HTMLElement>('panel')
  const presence = usePresence(open, () => panel.value)
  /** The panel while it is open, not while it leaves. */
  const live = computed(() => (presence.closing.value ? null : panel.value))

  const trigger = useTrigger(wrap, () => ({
    'aria-haspopup': 'dialog',
    'aria-expanded': open.value ? 'true' : 'false',
    'aria-controls': open.value ? id : undefined,
  }))
  const context = useOverlayContext(() => trigger.el.value ?? wrap.value)
  watch(open, (on) => on && context.refresh(), { immediate: true, flush: 'sync' })

  const close = () => (open.value = false)
  useDismiss(live, {
    onDismiss: close,
    escape: () => props.dismissible !== false,
    outside: false,
    returnTo: () => trigger.el.value,
  })
  useFocusTrap(live, {
    active: open,
    initial: () => {
      const el = live.value
      if (!el) return null
      const list = tabbables(el)
      return list.find((t) => !t.classList.contains(CLOSE_CLASS)) ?? list[0] ?? el
    },
  })
  useScrollLock(open)
  useInert(open)

  return {
    open,
    mounted,
    present: presence.present,
    closing: presence.closing,
    ids,
    panel,
    context,
    close,
    /** The trigger was pressed. */
    onTrigger(event: MouseEvent) {
      if (!event.defaultPrevented) open.value = true
    },
    /** The backdrop was pressed. */
    onBackdrop() {
      if (props.dismissible !== false) close()
    },
  }
}
