export interface CompanyFormat {
  timezone: string;
  currencyCode: string;
  currencyMinorDigits: number;
  locale?: string;
}

/** Format exact API amountMinor strings without ever converting the amount to Number. */
export function formatCurrencyMinor(amountMinor: string, config: CompanyFormat): string {
  if (!/^-?\d+$/.test(amountMinor)) throw new RangeError('Expected an integer decimal string.');
  const digits = config.currencyMinorDigits;
  if (!Number.isInteger(digits) || digits < 0 || digits > 20) throw new RangeError('Invalid currency precision.');
  if (!/^[A-Z]{3}$/.test(config.currencyCode)) throw new RangeError('Expected an ISO currency code.');
  const amount = BigInt(amountMinor);
  const absolute = amount < 0n ? -amount : amount;
  const scale = 10n ** BigInt(digits);
  const locale = config.locale ?? 'en';
  const currency = new Intl.NumberFormat(locale, {
    style: 'currency', currency: config.currencyCode, minimumFractionDigits: digits, maximumFractionDigits: digits,
  });
  const integer = new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(absolute / scale);
  const fraction = digits ? new Intl.NumberFormat(locale, {
    useGrouping: false, minimumIntegerDigits: digits, maximumFractionDigits: 0,
  }).format(absolute % scale) : '';
  // The small exemplar supplies locale-specific currency placement and the minus sign even for -0.01.
  let insertedInteger = false;
  return currency.formatToParts(amount < 0n ? -1 : 1).map(part => {
    if (part.type === 'integer') {
      if (insertedInteger) return '';
      insertedInteger = true;
      return integer;
    }
    if (part.type === 'group') return '';
    if (part.type === 'fraction') return fraction;
    return part.value;
  }).join('');
}

/** Date-only values are validated and displayed in UTC, never in the company or browser timezone. */
export function formatDateOnly(value: string, locale = 'en'): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) throw new RangeError('Expected YYYY-MM-DD.');
  const date = new Date(`${value}T00:00:00Z`);
  if (!Number.isFinite(date.getTime()) || date.toISOString().slice(0, 10) !== value) throw new RangeError('Invalid date.');
  return new Intl.DateTimeFormat(locale, { year: 'numeric', month: 'short', day: 'numeric', timeZone: 'UTC' }).format(date);
}

export function formatCompanyTimestamp(value: string, config: Pick<CompanyFormat, 'timezone' | 'locale'>): string {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value)) {
    throw new RangeError('Expected a timestamp with an explicit timezone.');
  }
  const date = new Date(value);
  // Date.parse normalizes some impossible calendar dates (for example February 30).
  formatDateOnly(value.slice(0, 10));
  if (!Number.isFinite(date.getTime())) throw new RangeError('Invalid timestamp.');
  return new Intl.DateTimeFormat(config.locale ?? 'en', {
    timeZone: config.timezone, year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit', timeZoneName: 'short',
  }).format(date);
}
