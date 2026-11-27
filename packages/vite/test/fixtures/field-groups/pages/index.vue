<script setup vapor lang="ts">
// A FormField's label over every control it can wrap, drawn by the server
// (field-groups.test.ts). Each `data-control` names the field's control and
// whether it is a labelable element (`native`) or names itself by the
// label's id (`named`).
import {
  Checkbox, CheckboxGroup, Combobox, DatePicker, DateRangePicker, FileUpload, FormField, Input, InputNumber,
  InputTags, InputTime, Listbox, PinInput, RadioGroup, Select, Slider, Switch, Textarea, Toggle, ToggleGroup,
  WeftProvider,
} from '@weft/ui'

const fruits = [
  { label: 'Apple', value: 'apple' },
  { label: 'Pear', value: 'pear' },
]
const sizes = [
  { label: 'S', value: 's' },
  { label: 'M', value: 'm' },
]
</script>

<template>
  <WeftProvider locale="en-GB" input="pointer">
    <div data-control="named RadioGroup"><FormField label="Plan" name="plan"><RadioGroup :items="['Starter', 'Team']" model-value="Team" /></FormField></div>
    <div data-control="named CheckboxGroup"><FormField label="Alerts" name="alerts"><CheckboxGroup :items="['Email', 'SMS']" :model-value="['Email']" /></FormField></div>
    <div data-control="named Slider"><FormField label="Volume" name="volume"><Slider :model-value="40" /></FormField></div>
    <div data-control="named Slider range"><FormField label="Price" name="price"><Slider :model-value="[20, 80]" /></FormField></div>
    <div data-control="named Listbox"><FormField label="Fruit" name="fruit"><Listbox :items="fruits" model-value="pear" /></FormField></div>
    <div data-control="named InputTime"><FormField label="At" name="at"><InputTime model-value="09:30" /></FormField></div>
    <div data-control="named InputTime range"><FormField label="Hours" name="hours"><InputTime range :model-value="{ start: '09:00', end: '17:00' }" /></FormField></div>
    <div data-control="named ToggleGroup"><FormField label="Size" name="size"><ToggleGroup :items="sizes" model-value="m" /></FormField></div>
    <div data-control="native Input"><FormField label="Email" name="email"><Input type="email" /></FormField></div>
    <div data-control="native Textarea"><FormField label="Bio" name="bio"><Textarea /></FormField></div>
    <div data-control="native InputNumber"><FormField label="Seats" name="seats"><InputNumber :model-value="2" /></FormField></div>
    <div data-control="native Checkbox"><FormField label="Terms" name="terms"><Checkbox label="I accept" /></FormField></div>
    <div data-control="native Switch"><FormField label="Dark" name="dark"><Switch label="On" /></FormField></div>
    <div data-control="native Toggle"><FormField label="Bold" name="bold"><Toggle label="B" /></FormField></div>
    <div data-control="native PinInput"><FormField label="Code" name="code"><PinInput :length="3" /></FormField></div>
    <div data-control="native Select"><FormField label="Pick" name="pick"><Select :items="fruits" model-value="apple" /></FormField></div>
    <div data-control="native Combobox"><FormField label="Search" name="search"><Combobox :items="fruits" /></FormField></div>
    <div data-control="native InputTags"><FormField label="Tags" name="tags"><InputTags :model-value="['vapor']" /></FormField></div>
    <div data-control="native Listbox filter"><FormField label="Find" name="find"><Listbox :items="fruits" filter /></FormField></div>
    <div data-control="native DatePicker"><FormField label="Due" name="due"><DatePicker model-value="2024-03-01" /></FormField></div>
    <div data-control="native DateRangePicker"><FormField label="Stay" name="stay"><DateRangePicker :model-value="{ start: '2024-04-02', end: '2024-04-05' }" /></FormField></div>
    <div data-control="native FileUpload"><FormField label="Files" name="files"><FileUpload /></FormField></div>
    <div data-control="native select"><FormField label="Country" id="country"><select id="country"><option>Kosovo</option></select></FormField></div>
  </WeftProvider>
  <WeftProvider locale="en-GB" input="touch">
    <div data-control="native InputTime touch"><FormField label="Wake" name="wake"><InputTime model-value="07:00" /></FormField></div>
    <div data-control="named InputTime touch range"><FormField label="Shift" name="shift"><InputTime range :model-value="{ start: '09:00', end: '17:00' }" /></FormField></div>
    <div data-control="native DatePicker touch"><FormField label="Born" name="born"><DatePicker model-value="2000-01-01" /></FormField></div>
  </WeftProvider>
</template>
