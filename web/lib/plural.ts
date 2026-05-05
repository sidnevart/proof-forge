/**
 * Russian plural form picker for cardinal numbers.
 *
 * Russian uses three plural forms:
 *   - one  → 1, 21, 31, …  ("1 участник")
 *   - few  → 2-4, 22-24, … ("2 участника")
 *   - many → 0, 5-20, 25-30, … ("5 участников")
 *
 * Special-case 11-14 which collapse to "many" despite ending in 1-4.
 */
export function pluralizeRu(n: number, forms: readonly [string, string, string]): string {
  const abs = Math.abs(Math.trunc(n));
  const mod10 = abs % 10;
  const mod100 = abs % 100;

  if (mod100 >= 11 && mod100 <= 14) return forms[2];
  if (mod10 === 1) return forms[0];
  if (mod10 >= 2 && mod10 <= 4) return forms[1];
  return forms[2];
}
