/**
 * useScrollLock(active)
 *
 * Stop the page behind an overlay from scrolling while `active` is true.
 * Locks are counted across the page: with a dialog and a menu from it both
 * locking, the page scrolls again only when the last of them lets go. The
 * first lock sets `overflow: hidden` on `<html>` and pads `<body>` by the
 * scrollbar's width so the page does not shift; the last release puts both
 * inline styles back exactly as they were. Each instance holds at most one
 * lock, and gives it up when its scope ends. Nothing happens on the server.
 */
import { onScopeDispose, watch, type Ref } from 'vue'

let holders = 0
let saved: { overflow: string; paddingRight: string } | null = null

function lock() {
  if (holders++ > 0) return
  const html = document.documentElement
  const body = document.body
  saved = { overflow: html.style.overflow, paddingRight: body.style.paddingRight }
  const bar = (html.ownerDocument.defaultView?.innerWidth ?? 0) - html.clientWidth
  if (bar > 0) body.style.paddingRight = `calc(${getComputedStyle(body).paddingRight} + ${bar}px)`
  html.style.overflow = 'hidden'
}

function unlock() {
  if (--holders > 0 || !saved) return
  document.documentElement.style.overflow = saved.overflow
  document.body.style.paddingRight = saved.paddingRight
  saved = null
}

export function useScrollLock(active: Readonly<Ref<boolean>>): void {
  let held = false
  const set = (on: boolean) => {
    if (on === held || typeof document === 'undefined') return
    held = on
    if (on) lock()
    else unlock()
  }
  watch(active, set, { immediate: true })
  onScopeDispose(() => set(false))
}
