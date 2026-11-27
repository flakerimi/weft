// The date and time controls EntityDialog draws for a field the schema says
// is a time: a DatePicker, and for a datetime a DatePicker and an InputTime
// side by side in a FieldGroup. A page pays for them, their CSS with the
// calendar's included, only when it imports useDateFields, since a page's
// CSS is all it can reach; without them EntityDialog draws the browser's own
// date and datetime inputs.
import { provide } from 'vue'
import DatePicker from '../dates/DatePicker.vue'
import FieldGroup from '../forms/FieldGroup.vue'
import { dateFieldsKey } from './entity.ts'
import FieldTime from './parts/FieldTime.vue'

/**
 * Gives every EntityDialog this component draws, a DataTable's among them,
 * the DatePicker and the InputTime for its date and datetime fields. Call it
 * in the setup of the island that draws them.
 */
export function useDateFields(): void {
  provide(dateFieldsKey, { DatePicker, FieldGroup, FieldTime })
}
