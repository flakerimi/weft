// Run before each test file. In the browser tests (happy-dom), a date
// picker's Popover and Calendar, which it loads on demand as a chunk of its
// own (@weft/ui's src/dates/picker-panel.ts), are taken as already fetched,
// as they are once a picker on the page has opened (see packages/ui's
// test/setup.ts). Under Node (renderPage) nothing is loaded.
if (typeof document !== 'undefined') await import('../../ui/src/dates/picker-panel.ts')

export {}
