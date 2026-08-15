import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const listeners = new Map<string, (...args: unknown[]) => void>();
    Object.assign(window, {
      runtime: {
        EventsOnMultiple: (
          name: string,
          callback: (...args: unknown[]) => void,
        ) => {
          listeners.set(name, callback);
          return () => listeners.delete(name);
        },
        EventsEmit: (name: string, ...args: unknown[]) =>
          listeners.get(name)?.(...args),
      },
      go: {
        main: {
          App: new Proxy(
            {},
            {
              get:
                (_target, name) =>
                async (...args: unknown[]) => {
                  const loaded = location.search.includes("loaded");
                  if (name === "Status")
                    return {
                      loaded,
                      changed: false,
                      total_pages: loaded ? 3 : 0,
                      total_blocks: loaded ? 3 : 0,
                      source: loaded ? "No assets" : "No source",
                    };
                  if (name === "Pages")
                    return {
                      pages: [0, 1, 2].map((index) => ({
                        index,
                        block_count: 1,
                        labels_summary: "text:1",
                      })),
                      page_from: 0,
                      page_to: 2,
                      total_pages: 3,
                    };
                  if (name === "View") {
                    const index = Number(args[0]);
                    await new Promise((resolve) =>
                      setTimeout(resolve, index === 1 ? 100 : 10),
                    );
                    return {
                      detail: {
                        page_index: index,
                        block_count: 1,
                        blocks: [
                          {
                            block_id: index,
                            label: "text",
                            preview: `page ${index}`,
                            order: "1",
                            bbox: "0, 0, 10, 10",
                          },
                        ],
                        image_url: null,
                        input_image: "",
                        is_remote_image: false,
                      },
                      layout: {
                        page_width: 100,
                        page_height: 100,
                        input_image: "/slow-page.png",
                        images: {},
                        blocks: [
                          {
                            block_id: index,
                            label: "text",
                            bbox: [0, 0, 10, 10],
                            content: `page ${index}`,
                            order: 1,
                          },
                        ],
                      },
                    };
                  }
                  if (name === "MoveBlock")
                    return { ok: true, message: "Moved block" };
                  if (name === "DeletePage")
                    return {
                      ok: true,
                      message: `Deleted page ${String(args[0])}`,
                      status: {
                        loaded: true,
                        changed: true,
                        total_pages: 2,
                        total_blocks: 2,
                        source: "No assets",
                      },
                    };
                  if (name === "SelectImportFile")
                    return "C:\\docs\\sample.json";
                  if (name === "Import") {
                    if (
                      args[0] !== "C:\\docs\\sample.json" ||
                      args[1] !== true
                    ) {
                      throw new Error("Import options were not forwarded");
                    }
                    if (location.search.includes("progress")) {
                      window.runtime.EventsEmit("task:progress", {
                        task: "import",
                        done: 2,
                        total: 10,
                        stage: "Downloading images",
                      });
                      await new Promise((resolve) => setTimeout(resolve, 300));
                    }
                    return { total_pages: 3, total_blocks: 3 };
                  }
                  if (name === "CancelTask") return undefined;
                  if (name === "ExportMarkdown") {
                    const request = args[0] as { start: number; end: number };
                    if (request.start !== 1 || request.end !== 3) {
                      throw new Error(
                        "Markdown range was not set to all pages",
                      );
                    }
                    return "all-pages.md";
                  }
                  if (name === "Search") {
                    const content = String(
                      (args[0] as { content?: string | null }).content ?? "",
                    );
                    await new Promise((resolve) =>
                      setTimeout(resolve, content === "old" ? 100 : 10),
                    );
                    return {
                      hits: [
                        {
                          page_index: 0,
                          block_id: content === "old" ? 1 : 2,
                          label: "text",
                          preview: content,
                        },
                      ],
                      total: 1,
                    };
                  }
                  if (name === "SetPageToolsVisible") return undefined;
                  throw new Error(`Unexpected binding: ${String(name)}`);
                },
            },
          ),
        },
      },
    });
  });
});

test("starts the Svelte renderer with the Wails boundary", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("http://127.0.0.1:4173");
  await expect(
    page.getByRole("dialog", { name: "Import & index JSON" }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Select file" })).toBeFocused();
  await expect(
    page.getByRole("button", { name: "Import", exact: true }),
  ).toBeDisabled();
  await expect(page.getByRole("heading", { name: "Page tools" })).toHaveCount(
    0,
  );
  await expect(page.getByText("No source", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect(
    page.getByRole("dialog", { name: "Import & index JSON" }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Import JSON" }).click();
  await expect(
    page.getByRole("dialog", { name: "Import & index JSON" }),
  ).toBeVisible();
  expect(errors).toEqual([]);
});

test("selects a file and options before importing", async ({ page }) => {
  await page.goto("http://127.0.0.1:4173");
  const dialog = page.getByRole("dialog", { name: "Import & index JSON" });

  await dialog.getByRole("button", { name: "Select file" }).click();
  await expect(dialog.getByText("sample.json", { exact: true })).toBeVisible();
  await dialog.getByText("Download remote images").click();
  await expect(dialog.getByRole("checkbox")).toBeChecked();
  await dialog.getByRole("button", { name: "Import" }).click();

  await expect(dialog).toHaveCount(0);
  await expect(page.getByText("Indexed 3 pages (3 blocks)")).toBeVisible();
});

test("shows import progress and locks the modal", async ({ page }) => {
  await page.goto("http://127.0.0.1:4173/?progress");
  const dialog = page.getByRole("dialog", { name: "Import & index JSON" });
  await dialog.getByRole("button", { name: "Select file" }).click();
  await dialog.getByText("Download remote images").click();
  await dialog.getByRole("button", { name: "Import", exact: true }).click();

  await expect(dialog.getByText("Downloading images (2 / 10)")).toBeVisible();
  await expect(dialog.getByRole("progressbar")).toHaveAttribute("value", "2");
  await expect(dialog.getByRole("button", { name: "Close" })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeVisible();
});

test("keeps the page visible and applies only the latest selection", async ({
  page,
}) => {
  await page.route("**/slow-page.png", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 500));
    await route.abort();
  });
  await page.goto("http://127.0.0.1:4173/?loaded");
  await page.getByRole("button", { name: /^p0/ }).click();
  await expect(page.getByRole("heading", { name: "Page 0" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: /#0 text .*page 0/ }).first(),
  ).toBeVisible();

  await page.getByRole("button", { name: /^p1/ }).click();
  await page.getByRole("button", { name: /^p2/ }).click();
  await expect(page.getByText("Loading page…")).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Page 2" })).toBeVisible();
  await page.waitForTimeout(120);
  await expect(page.getByRole("heading", { name: "Page 2" })).toBeVisible();
});

test("keeps page, block, and canvas interactions intact", async ({ page }) => {
  await page.route("**/slow-page.png", (route) =>
    route.fulfill({
      contentType: "image/svg+xml",
      body: '<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect width="100" height="100" fill="white"/></svg>',
    }),
  );
  await page.goto("http://127.0.0.1:4173/?loaded");
  const pageButton = page.getByRole("button", { name: /^p0/ });
  await pageButton.click();

  const canvas = page.getByRole("application", { name: "Page canvas" });
  await canvas.focus();
  await canvas.hover();
  await page.mouse.wheel(0, -100);
  await expect(page.getByText("110%", { exact: true })).toBeVisible();

  const block = page.locator("[data-block-id='0']");
  await expect(block).toBeVisible();
  await expect(block).toHaveCSS("cursor", "pointer");
  const box = await block.boundingBox();
  if (!box) throw new Error("Block overlay has no bounding box");
  await page.keyboard.down("Control");
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 + 8, box.y + box.height / 2 + 8);
  await page.mouse.up();
  await page.keyboard.up("Control");
  await expect(page.getByText("Loading image…")).toHaveCount(0);

  await block.click();
  await expect(page.getByRole("heading", { name: "Block #0" })).toBeVisible();

  const blockList = page
    .getByRole("heading", { name: "Blocks" })
    .locator("..")
    .locator("div")
    .first();
  expect(
    await blockList.evaluate(
      (element) => element.scrollHeight <= element.clientHeight,
    ),
  ).toBe(true);
  await expect(page.getByRole("switch").locator("..")).toHaveCSS(
    "border-top-width",
    "0px",
  );
});

test("confirms page deletion", async ({ page }) => {
  await page.goto("http://127.0.0.1:4173/?loaded");
  await page.getByRole("button", { name: /^p0/ }).click();
  await page.evaluate(() =>
    window.runtime.EventsEmit("menu:command", "page-tools"),
  );
  await page.getByRole("button", { name: "Delete page" }).click();
  const confirm = page.getByRole("dialog", { name: "Delete page 0?" });
  await expect(confirm).toBeVisible();
  await confirm.getByRole("button", { name: "Delete page" }).click();
  await expect(page.getByText("Deleted page 0")).toBeVisible();
});

test("keeps the all-pages range while a page finishes loading", async ({
  page,
}) => {
  await page.goto("http://127.0.0.1:4173/?loaded");
  await page.getByRole("button", { name: /^p1/ }).click();
  await page.evaluate(() =>
    window.runtime.EventsEmit("menu:command", "export-markdown"),
  );
  const dialog = page.getByRole("dialog", { name: "Export Markdown" });
  await dialog.getByRole("button", { name: "All pages" }).click();
  await page.waitForTimeout(120);
  await expect(dialog.getByLabel("Start page")).toHaveValue("1");
  await expect(dialog.getByLabel("End page")).toHaveValue("3");
  await dialog.getByRole("button", { name: "Export", exact: true }).click();
  await expect(page.getByText("Exported all-pages.md")).toBeVisible();
});

test("keeps only the latest search response", async ({ page }) => {
  await page.goto("http://127.0.0.1:4173/?loaded");
  await page.evaluate(() =>
    window.runtime.EventsEmit("menu:command", "search"),
  );
  const form = page
    .getByRole("dialog", { name: "Search blocks" })
    .locator("form");
  const content = form.getByLabel("Content contains");

  await content.fill("old");
  await form.evaluate((element) =>
    element.dispatchEvent(
      new SubmitEvent("submit", { bubbles: true, cancelable: true }),
    ),
  );
  await content.fill("new");
  await form.evaluate((element) =>
    element.dispatchEvent(
      new SubmitEvent("submit", { bubbles: true, cancelable: true }),
    ),
  );

  await expect(page.getByText("new", { exact: true })).toBeVisible();
  await page.waitForTimeout(120);
  await expect(page.getByText("new", { exact: true })).toBeVisible();
  await expect(page.getByText("old", { exact: true })).toHaveCount(0);
});
