import { test, expect } from '@playwright/test';

test.describe('Minimal Test', () => {
    test('should pass', async () => {
        expect(true).toBe(true);
    });
});
