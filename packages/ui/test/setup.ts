// Run before each test file. A date picker loads what it draws while open,
// its Popover and Calendar (src/dates/picker-panel.ts), on demand, as a chunk
// of its own. Here it is taken as already fetched, as it is once a picker on
// the page has opened, so a test that opens one does not wait on the module
// loader. test/dates/lazy-panel.test.ts opens one while it is still on its way.
import '../src/dates/picker-panel.ts'
