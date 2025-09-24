import { describe, it, expect } from 'vitest'
import { formatDate, formatBytes, formatUptime, truncateText } from '@/utils/format'

describe('Format Utils', () => {
  describe('formatDate', () => {
    it('formats ISO date string correctly', () => {
      const isoDate = '2023-01-15T10:30:00Z'
      const result = formatDate(isoDate)
      expect(result).toMatch(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)
    })

    it('handles null/undefined input', () => {
      expect(formatDate(null)).toBe('')
      expect(formatDate(undefined)).toBe('')
      expect(formatDate('')).toBe('')
    })

    it('handles invalid date string', () => {
      const invalidDate = 'invalid-date'
      const result = formatDate(invalidDate)
      expect(result).toBe('')
    })

    it('formats different date formats', () => {
      const timestamp = Date.now()
      const result = formatDate(timestamp)
      expect(result).toMatch(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)
    })

    it('respects custom format parameter', () => {
      const isoDate = '2023-01-15T10:30:00Z'
      const result = formatDate(isoDate, 'YYYY-MM-DD')
      expect(result).toBe('2023-01-15')
    })
  })

  describe('formatBytes', () => {
    it('formats bytes correctly', () => {
      expect(formatBytes(0)).toBe('0 Bytes')
      expect(formatBytes(1)).toBe('1 Bytes')
      expect(formatBytes(1024)).toBe('1 KB')
      expect(formatBytes(1048576)).toBe('1 MB')
      expect(formatBytes(1073741824)).toBe('1 GB')
      expect(formatBytes(1099511627776)).toBe('1 TB')
    })

    it('handles decimal places correctly', () => {
      expect(formatBytes(1500)).toBe('1.46 KB')
      expect(formatBytes(1536)).toBe('1.5 KB')
      expect(formatBytes(1600000)).toBe('1.53 MB')
    })

    it('handles negative numbers', () => {
      expect(formatBytes(-1024)).toBe('-1 KB')
    })

    it('handles null/undefined input', () => {
      expect(formatBytes(null)).toBe('0 Bytes')
      expect(formatBytes(undefined)).toBe('0 Bytes')
    })

    it('respects custom decimals parameter', () => {
      expect(formatBytes(1536, 0)).toBe('2 KB')
      expect(formatBytes(1536, 3)).toBe('1.5 KB')
    })
  })

  describe('formatUptime', () => {
    it('formats seconds correctly', () => {
      expect(formatUptime(0)).toBe('0s')
      expect(formatUptime(30)).toBe('30s')
      expect(formatUptime(59)).toBe('59s')
    })

    it('formats minutes correctly', () => {
      expect(formatUptime(60)).toBe('1m 0s')
      expect(formatUptime(90)).toBe('1m 30s')
      expect(formatUptime(3599)).toBe('59m 59s')
    })

    it('formats hours correctly', () => {
      expect(formatUptime(3600)).toBe('1h 0m 0s')
      expect(formatUptime(3661)).toBe('1h 1m 1s')
      expect(formatUptime(86399)).toBe('23h 59m 59s')
    })

    it('formats days correctly', () => {
      expect(formatUptime(86400)).toBe('1d 0h 0m 0s')
      expect(formatUptime(90061)).toBe('1d 1h 1m 1s')
      expect(formatUptime(172800)).toBe('2d 0h 0m 0s')
    })

    it('handles negative numbers', () => {
      expect(formatUptime(-60)).toBe('-1m 0s')
    })

    it('handles null/undefined input', () => {
      expect(formatUptime(null)).toBe('0s')
      expect(formatUptime(undefined)).toBe('0s')
    })

    it('handles very large numbers', () => {
      const largeNumber = 31536000 // 1 year in seconds
      const result = formatUptime(largeNumber)
      expect(result).toBe('365d 0h 0m 0s')
    })
  })

  describe('truncateText', () => {
    it('does not truncate text shorter than limit', () => {
      const text = 'Short text'
      expect(truncateText(text, 20)).toBe('Short text')
    })

    it('truncates text longer than limit', () => {
      const text = 'This is a very long text that should be truncated'
      expect(truncateText(text, 20)).toBe('This is a very lon...')
    })

    it('uses custom suffix', () => {
      const text = 'This is a very long text that should be truncated'
      expect(truncateText(text, 20, ' [more]')).toBe('This is a very [more]')
    })

    it('handles empty string', () => {
      expect(truncateText('', 20)).toBe('')
    })

    it('handles null/undefined input', () => {
      expect(truncateText(null, 20)).toBe('')
      expect(truncateText(undefined, 20)).toBe('')
    })

    it('handles zero or negative limit', () => {
      const text = 'Some text'
      expect(truncateText(text, 0)).toBe('...')
      expect(truncateText(text, -5)).toBe('...')
    })

    it('handles text exactly at limit', () => {
      const text = 'Exactly 20 chars!!'
      expect(truncateText(text, 20)).toBe('Exactly 20 chars!!')
    })

    it('preserves word boundaries when possible', () => {
      const text = 'This is a sentence with several words'
      expect(truncateText(text, 20)).toBe('This is a sentence...')
    })
  })
})
