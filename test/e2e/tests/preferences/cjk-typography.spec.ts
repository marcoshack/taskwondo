import { test, expect } from '../../lib/fixtures';

async function attach(page: any, testInfo: any, name: string) {
  const screenshot = await page.screenshot();
  await testInfo.attach(name, { body: screenshot, contentType: 'image/png' });
}

async function rootFontSize(page: any): Promise<number> {
  return page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));
}

async function selectLanguage(page: any, nativeLabel: string, lang: string) {
  await page.goto('/preferences/appearance');
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: new RegExp(`^${nativeLabel}\\s`) }).click();
  await expect.poll(() => page.evaluate(() => document.documentElement.lang)).toBe(lang);
}

async function selectFontSize(page: any, label: string, value: string) {
  await page.getByRole('button', { name: new RegExp(`^${label}`) }).click();
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem('taskwondo_font_size')))
    .toBe(value);
}

test.describe('CJK typography', () => {
  test('chinese ui renders at an optically smaller size and uses a CJK font stack', async ({ page }, testInfo) => {
    await selectLanguage(page, 'English', 'en');
    const latinSize = await rootFontSize(page);
    expect(latinSize).toBeCloseTo(17.6, 1);
    expect(await page.evaluate(() => getComputedStyle(document.body).fontFamily)).not.toContain('PingFang SC');

    await selectLanguage(page, '中文', 'zh');
    const zhSize = await rootFontSize(page);
    // CJK glyphs fill the em box, so the root size is nudged down ~9% for zh
    expect(zhSize).toBeLessThan(latinSize - 1);
    expect(zhSize).toBeCloseTo(latinSize * 0.909, 0);
    expect(await page.evaluate(() => getComputedStyle(document.body).fontFamily)).toContain('PingFang SC');
    await attach(page, testInfo, '01-chinese-appearance');

    await selectLanguage(page, 'English', 'en');
    expect(await rootFontSize(page)).toBeCloseTo(latinSize, 1);
  });

  test('font size preference still scales the whole ui for chinese', async ({ page }, testInfo) => {
    await selectLanguage(page, '中文', 'zh');
    const normal = await rootFontSize(page);

    await selectFontSize(page, '大', 'large');
    const larger = await rootFontSize(page);
    expect(larger).toBeGreaterThan(normal);
    await attach(page, testInfo, '02-chinese-larger');

    await selectFontSize(page, '小', 'small');
    const smaller = await rootFontSize(page);
    expect(smaller).toBeLessThan(normal);
    expect(smaller).toBeCloseTo(16 * 0.909, 0);
    await attach(page, testInfo, '03-chinese-smaller');
  });
});
