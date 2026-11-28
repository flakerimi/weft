import { execFileSync } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { describe, expect, it } from 'vitest'
import { dateToWire, wireTimeText, wireToDate, wireToLocal } from '../../src/data/when.ts'

// A browser's zone is the process's, fixed when it starts: V8 does not take a
// TZ set later. So each zone's cases run in a Node of their own, started with
// TZ set before any Date is made, which loads when.ts as it is.
const when = pathToFileURL(join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'src', 'data', 'when.ts')).href
function inZone<T>(tz: string, expression: string): T {
  const source = `import { dateToWire, localToWire, wireTimeText, wireToDate, wireToLocal } from ${JSON.stringify(when)}
console.log(JSON.stringify(${expression}))`
  const out = execFileSync(process.execPath, ['--input-type=module', '-e', source], {
    env: { ...process.env, TZ: tz },
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  })
  return JSON.parse(out) as T
}

const ZONES = ['Pacific/Pago_Pago', 'Pacific/Kiritimati', 'Europe/Berlin'] // UTC-11, UTC+14, and a day the clocks change there
// Narrow no-break spaces, which newer ICU puts before AM and PM, read as spaces.
const plain = (s: string) => s.replace(/[  ]/g, ' ')

describe('when', () => {
  for (const tz of ZONES) {
    it(`keeps a date as written in ${tz}`, () => {
      expect(
        inZone(tz, `[
          wireToDate('2026-03-29T00:00:00Z'),
          dateToWire('2026-03-29'),
          dateToWire(wireToDate('2026-03-29T00:00:00Z')),
          wireToDate(dateToWire('2026-03-29')),
          wireTimeText('2026-03-29T00:00:00Z', 'date', 'en-US'),
        ]`),
      ).toEqual(['2026-03-29', '2026-03-29T00:00:00Z', '2026-03-29T00:00:00Z', '2026-03-29', 'Mar 29, 2026'])
    })

    it(`keeps a datetime's instant in ${tz}`, () => {
      const wires = ['2026-03-29T01:30:00Z', '2026-03-28T23:45:00Z', '2026-10-25T00:30:00Z', '2026-12-31T10:00:00Z', '2026-01-01T00:00:00Z']
      const back = inZone<string[]>(tz, `${JSON.stringify(wires)}.map((w) => { const l = wireToLocal(w); return localToWire(l.date, l.time) })`)
      expect(back).toEqual(wires)
    })
  }

  it('shows a datetime in the zone the browser is in', () => {
    const at = `wireToLocal('2026-03-29T01:30:00Z')`
    expect(inZone('Pacific/Pago_Pago', at)).toEqual({ date: '2026-03-28', time: '14:30' })
    expect(inZone('Pacific/Kiritimati', at)).toEqual({ date: '2026-03-29', time: '15:30' })
    expect(
      inZone('Europe/Berlin', `[${at}, wireToLocal('2026-03-29T00:30:00Z'), localToWire('2026-03-29', '01:30'), localToWire('2026-10-25', '12:00')]`),
    ).toEqual([{ date: '2026-03-29', time: '03:30' }, { date: '2026-03-29', time: '01:30' }, '2026-03-29T00:30:00Z', '2026-10-25T11:00:00Z'])
    expect(inZone('Pacific/Pago_Pago', `localToWire('2026-03-28', '14:30')`)).toBe('2026-03-29T01:30:00Z')
    expect(inZone('Pacific/Kiritimati', `localToWire('2026-03-29', '15:30')`)).toBe('2026-03-29T01:30:00Z')
  })

  it('moves a time the clocks skip forward by the hour they skip', () => {
    // 02:30 on 2026-03-29 does not happen in Berlin: the clocks go from 02:00 to 03:00.
    expect(inZone('Europe/Berlin', `[localToWire('2026-03-29', '02:30'), wireToLocal(localToWire('2026-03-29', '02:30'))]`)).toEqual([
      '2026-03-29T01:30:00Z',
      { date: '2026-03-29', time: '03:30' },
    ])
  })

  it('writes a column’s date or datetime in a locale', () => {
    expect(inZone('UTC', `wireTimeText('2026-03-29T00:00:00Z', 'date', 'en-GB')`)).toBe('29 Mar 2026')
    expect(plain(inZone('UTC', `wireTimeText('2026-03-29T13:30:00Z', 'datetime', 'en-US')`))).toBe('Mar 29, 2026, 1:30 PM')
    expect(plain(inZone('Pacific/Kiritimati', `wireTimeText('2026-03-29T13:30:00Z', 'datetime', 'en-US')`))).toBe('Mar 30, 2026, 3:30 AM')
    expect(inZone('Pacific/Pago_Pago', `wireTimeText('2026-03-29T00:00:00Z', 'date', 'en-US')`)).toBe('Mar 29, 2026')
  })

  it('keeps a year before 100 as it is', () => {
    expect(inZone('UTC', `[localToWire('0050-06-01', '12:00'), wireToLocal('0050-06-01T12:00:00Z')]`)).toEqual([
      '0050-06-01T12:00:00Z',
      { date: '0050-06-01', time: '12:00' },
    ])
  })

  it('reads Go’s zero time and nothing as null', () => {
    for (const empty of ['0001-01-01T00:00:00Z', '', null, undefined]) {
      expect(wireToDate(empty)).toBeNull()
      expect(wireToLocal(empty)).toBeNull()
      expect(wireTimeText(empty, 'date')).toBe('')
      expect(wireTimeText(empty, 'datetime')).toBe('')
    }
    expect(dateToWire(null)).toBeNull()
    expect(dateToWire('')).toBeNull()
    expect(dateToWire('29/03/2026')).toBeNull()
    expect(wireToDate('soon')).toBeNull()
    expect(wireToLocal('soon')).toBeNull()
    expect(wireTimeText('soon', 'datetime')).toBe('')
    expect(wireTimeText(42, 'date')).toBe('')
    expect(inZone('UTC', `[localToWire('soon', '12:00'), localToWire('2026-03-29', 'noon')]`)).toEqual(['', ''])
  })
})
