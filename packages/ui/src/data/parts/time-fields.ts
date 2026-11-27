// What EntityDialog draws only for a field the schema says is a time, loaded
// on demand as one chunk the first time a dialog has such a field, so a
// dialog (and a DataTable) without one does not carry them: the DatePicker,
// the FieldGroup a datetime's date and time sit in, and the time.
export { default as DatePicker } from '../../dates/DatePicker.vue'
export { default as FieldGroup } from '../../forms/FieldGroup.vue'
export { default as FieldTime } from './FieldTime.vue'
