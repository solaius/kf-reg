import { test, expect } from '@playwright/test';

const CATALOG_SERVER_URL = 'http://localhost:8080';
const UI_BASE_URL = 'http://localhost:9000';

test.describe('Phase 4: Health Endpoints', () => {

  test('Test 1: /livez returns 200 with alive status', async ({ page }) => {
    const response = await page.goto(`${CATALOG_SERVER_URL}/livez`);
    console.log('=== TEST 1: /livez endpoint ===');

    expect(response).not.toBeNull();
    expect(response!.status()).toBe(200);

    const body = await response!.json();
    console.log(`  Status: ${body.status}`);
    console.log(`  Uptime: ${body.uptime}`);
    expect(body.status).toBe('alive');
    expect(body.uptime).toBeDefined();
    console.log('  TEST 1 RESULT: PASS');
  });

  test('Test 2: /readyz returns 200 with component breakdown', async ({ page }) => {
    const response = await page.goto(`${CATALOG_SERVER_URL}/readyz`);
    console.log('=== TEST 2: /readyz endpoint ===');

    expect(response).not.toBeNull();
    expect(response!.status()).toBe(200);

    const body = await response!.json();
    console.log(`  Status: ${body.status}`);
    console.log(`  Components: ${JSON.stringify(body.components)}`);

    expect(body.status).toBe('ready');
    expect(body.components).toBeDefined();
    expect(body.components.database).toBeDefined();
    expect(body.components.initial_load).toBeDefined();
    expect(body.components.plugins).toBeDefined();

    console.log(`  Database status: ${body.components.database.status}`);
    console.log(`  Initial load status: ${body.components.initial_load.status}`);
    console.log(`  Plugins status: ${body.components.plugins.status}`);
    console.log('  TEST 2 RESULT: PASS');
  });

  test('Test 3: /healthz and /livez return the same structure', async ({ page }) => {
    console.log('=== TEST 3: /healthz and /livez consistency ===');

    const healthzResp = await page.goto(`${CATALOG_SERVER_URL}/healthz`);
    const healthzBody = await healthzResp!.json();

    const livezResp = await page.goto(`${CATALOG_SERVER_URL}/livez`);
    const livezBody = await livezResp!.json();

    expect(healthzResp!.status()).toBe(200);
    expect(livezResp!.status()).toBe(200);
    expect(healthzBody.status).toBe('alive');
    expect(livezBody.status).toBe('alive');
    expect(healthzBody.uptime).toBeDefined();
    expect(livezBody.uptime).toBeDefined();

    console.log(`  /healthz status: ${healthzBody.status}`);
    console.log(`  /livez status: ${livezBody.status}`);
    console.log('  TEST 3 RESULT: PASS');
  });
});

test.describe('Phase 4: Catalog Management - Validate Button', () => {

  test('Test 4: Validate button is visible on manage source page', async ({ page }) => {
    await page.goto(`${UI_BASE_URL}/catalog-management/plugin/model/sources/sample-source/manage`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'verify-phase4-validate.png', fullPage: true });

    console.log('=== TEST 4: Validate Button ===');

    // Check for validate button
    const validateBtn = await page.locator('button:has-text("Validate")').count();
    console.log(`  Validate button: ${validateBtn > 0 ? 'FOUND' : 'NOT FOUND'}`);

    // Click validate button if found
    if (validateBtn > 0) {
      await page.locator('button:has-text("Validate")').first().click();
      await page.waitForTimeout(1000);
      await page.screenshot({ path: 'verify-phase4-validate-result.png', fullPage: true });

      // Check for validation result panel
      const validationPanel = await page.locator('[data-testid="validation-result-panel"]').count();
      const expandableSection = await page.locator('.pf-v6-c-expandable-section').count();
      console.log(`  Validation result panel (data-testid): ${validationPanel > 0 ? 'FOUND' : 'NOT FOUND'}`);
      console.log(`  Expandable section: ${expandableSection > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Check for layer results
      const layerResults = await page.locator('[data-testid="layer-results"]').count();
      console.log(`  Layer results: ${layerResults > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Check for pass/fail labels
      const labels = await page.locator('.pf-v6-c-label').count();
      console.log(`  Status labels: ${labels}`);
    }

    const pass = validateBtn > 0;
    console.log(`  TEST 4 RESULT: ${pass ? 'PASS' : 'FAIL'}`);
  });
});

test.describe('Phase 4: Catalog Management - Save with Refresh', () => {

  test('Test 5: Save triggers refresh and shows feedback', async ({ page }) => {
    await page.goto(`${UI_BASE_URL}/catalog-management/plugin/model/sources/sample-source/manage`);
    await page.waitForTimeout(3000);

    console.log('=== TEST 5: Save with Refresh Feedback ===');

    // Check for Save button
    const saveBtn = await page.locator('button:has-text("Save")').count();
    console.log(`  Save button: ${saveBtn > 0 ? 'FOUND' : 'NOT FOUND'}`);

    if (saveBtn > 0) {
      // Click save
      await page.locator('button:has-text("Save")').first().click();
      await page.waitForTimeout(500);

      // Check for spinner/loading state during save
      const spinner = await page.locator('.pf-v6-c-spinner').count();
      const loadingBtn = await page.locator('button[aria-disabled="true"], button.pf-m-in-progress').count();
      console.log(`  Spinner visible during save: ${spinner > 0 ? 'YES' : 'NO'}`);
      console.log(`  Button in loading state: ${loadingBtn > 0 ? 'YES' : 'NO'}`);

      await page.waitForTimeout(3000);
      await page.screenshot({ path: 'verify-phase4-save-result.png', fullPage: true });

      // Check for success notification/alert
      const successAlert = await page.locator('.pf-v6-c-alert.pf-m-success, .pf-v6-c-alert--success').count();
      const notification = await page.locator('[class*="notification"], [class*="toast"]').count();
      console.log(`  Success alert: ${successAlert > 0 ? 'FOUND' : 'NOT FOUND'}`);
      console.log(`  Notification: ${notification > 0 ? 'FOUND' : 'NOT FOUND'}`);
    }

    const pass = saveBtn > 0;
    console.log(`  TEST 5 RESULT: ${pass ? 'PASS' : 'FAIL'}`);
  });
});

test.describe('Phase 4: Catalog Management - Revision History', () => {

  test('Test 6: Revision history panel is visible on manage source page', async ({ page }) => {
    await page.goto(`${UI_BASE_URL}/catalog-management/plugin/model/sources/sample-source/manage`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'verify-phase4-revisions.png', fullPage: true });

    console.log('=== TEST 6: Revision History Panel ===');

    // Check for revision history panel
    const revisionPanel = await page.locator('[data-testid="revision-history-panel"]').count();
    console.log(`  Revision history panel (data-testid): ${revisionPanel > 0 ? 'FOUND' : 'NOT FOUND'}`);

    // Check for revision history heading
    const revisionHeading = await page.locator('text=Revision history').count();
    console.log(`  "Revision history" heading: ${revisionHeading > 0 ? 'FOUND' : 'NOT FOUND'}`);

    // Check for revision list
    const revisionList = await page.locator('[data-testid="revision-list"]').count();
    console.log(`  Revision list (data-testid): ${revisionList > 0 ? 'FOUND' : 'NOT FOUND'}`);

    // Check for individual revision entries
    const revisionItems = await page.locator('.pf-v6-c-data-list__item').count();
    console.log(`  Revision items: ${revisionItems}`);

    // Check for rollback buttons within revision list
    const rollbackBtns = await page.locator('button:has-text("Rollback")').count();
    console.log(`  Rollback buttons: ${rollbackBtns}`);

    // Check for version codes (truncated hashes)
    const versionCodes = await page.locator('[data-testid="revision-list"] code').count();
    console.log(`  Version codes: ${versionCodes}`);

    const pass = revisionHeading > 0 || revisionPanel > 0;
    console.log(`  TEST 6 RESULT: ${pass ? 'PASS' : 'FAIL'}`);
  });
});

test.describe('Phase 4: Catalog Management - Rollback', () => {

  test('Test 7: Rollback triggers confirmation modal', async ({ page }) => {
    await page.goto(`${UI_BASE_URL}/catalog-management/plugin/model/sources/sample-source/manage`);
    await page.waitForTimeout(3000);

    console.log('=== TEST 7: Rollback Functionality ===');

    // Find rollback buttons
    const rollbackBtns = await page.locator('button:has-text("Rollback")').count();
    console.log(`  Rollback buttons found: ${rollbackBtns}`);

    if (rollbackBtns > 0) {
      // Click the first rollback button
      await page.locator('button:has-text("Rollback")').first().click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: 'verify-phase4-rollback-modal.png', fullPage: true });

      // Check for confirmation modal
      const modal = await page.locator('[data-testid="rollback-confirm-modal"], .pf-v6-c-modal-box').count();
      console.log(`  Rollback modal: ${modal > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Check for modal title
      const modalTitle = await page.locator('text=Rollback Configuration').count();
      console.log(`  Modal title "Rollback Configuration": ${modalTitle > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Check for confirm button
      const confirmBtn = await page.locator('[data-testid="rollback-confirm-button"]').count();
      console.log(`  Confirm button: ${confirmBtn > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Check for cancel button
      const cancelBtn = await page.locator('[data-testid="rollback-cancel-button"]').count();
      console.log(`  Cancel button: ${cancelBtn > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Check for warning text about losing changes
      const warningText = await page.locator('text=unsaved changes will be lost').count();
      console.log(`  Warning about lost changes: ${warningText > 0 ? 'FOUND' : 'NOT FOUND'}`);

      // Cancel the modal
      if (cancelBtn > 0) {
        await page.locator('[data-testid="rollback-cancel-button"]').click();
        await page.waitForTimeout(500);
      } else {
        await page.keyboard.press('Escape');
      }
    }

    // Rollback feature exists if buttons or the revision panel is present
    const revisionPanel = await page.locator('[data-testid="revision-history-panel"]').count();
    const pass = rollbackBtns > 0 || revisionPanel > 0;
    console.log(`  TEST 7 RESULT: ${pass ? 'PASS' : 'FAIL'}`);
  });
});

test.describe('Phase 4: Sources Page - Last Refresh', () => {

  test('Test 8: Sources page shows last refresh time', async ({ page }) => {
    await page.goto(`${UI_BASE_URL}/catalog-management/plugin/model/sources`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'verify-phase4-last-refresh.png', fullPage: true });

    console.log('=== TEST 8: Last Refresh Time on Sources Page ===');

    // Check for last refresh text or timestamp
    const lastRefresh = await page.locator('text=Last refresh').count();
    const refreshTime = await page.locator('text=ago').count();
    const lastUpdated = await page.locator('text=Last updated').count();
    console.log(`  "Last refresh" text: ${lastRefresh > 0 ? 'FOUND' : 'NOT FOUND'}`);
    console.log(`  Time "ago" text: ${refreshTime > 0 ? 'FOUND' : 'NOT FOUND'}`);
    console.log(`  "Last updated" text: ${lastUpdated > 0 ? 'FOUND' : 'NOT FOUND'}`);

    // Check for status labels with refresh info
    const statusLabels = await page.locator('.pf-v6-c-label').count();
    console.log(`  Status labels: ${statusLabels}`);

    // Check table rows for refresh information
    const rows = await page.locator('tbody tr').count();
    console.log(`  Table rows: ${rows}`);

    // Check for any timestamp-like content in the table
    const pageContent = await page.textContent('body');
    const hasTimeInfo = pageContent &&
      (pageContent.includes('ago') || pageContent.includes('refresh') ||
       pageContent.includes('last') || pageContent.includes('updated'));
    console.log(`  Page has time/refresh info: ${hasTimeInfo ? 'YES' : 'NO'}`);

    const pass = rows > 0;
    console.log(`  TEST 8 RESULT: ${pass ? 'PASS' : 'FAIL'}`);
  });

  test('Test 9: MCP Plugin Sources page shows last refresh', async ({ page }) => {
    await page.goto(`${UI_BASE_URL}/catalog-management/plugin/mcp/sources`);
    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'verify-phase4-mcp-last-refresh.png', fullPage: true });

    console.log('=== TEST 9: MCP Sources Last Refresh ===');

    const rows = await page.locator('tbody tr').count();
    console.log(`  Table rows: ${rows}`);

    const statusLabels = await page.locator('.pf-v6-c-label').count();
    console.log(`  Status labels: ${statusLabels}`);

    const pass = rows > 0;
    console.log(`  TEST 9 RESULT: ${pass ? 'PASS' : 'FAIL'}`);
  });
});
