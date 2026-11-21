// What a date picker draws only while it is open, loaded on demand as one
// chunk (useLazyPanel in picker.ts): the Popover it opens (with the floating
// positioning, the overlay stack and the touch sheet) and the Calendar
// inside it (its grid, its month and year pickers, their formatting).
export { default as Popover } from '../overlays/Popover.vue'
export { default as Calendar } from './Calendar.vue'
