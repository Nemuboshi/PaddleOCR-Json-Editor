<script lang="ts">
import { onMount } from "svelte";
import { api } from "./api";
import ImportModal from "./features/dialogs/ImportModal.svelte";
import MarkdownExportModal from "./features/dialogs/MarkdownExportModal.svelte";
import SearchModal from "./features/dialogs/SearchModal.svelte";
import UnsavedDialog from "./features/dialogs/UnsavedDialog.svelte";
import BlockEditor from "./features/editor/BlockEditor.svelte";
import PagePanel from "./features/editor/PagePanel.svelte";
import PageList from "./features/pages/PageList.svelte";
import PageTools from "./features/pages/PageTools.svelte";
import { formatErrorText } from "./copy";
import { type EditorAnchor, viewportCenterAnchor } from "./lib/anchor";
import {
  type ErrorPresentation,
  presentCommandError,
} from "./lib/errors";
import appStyles from "./styles/app.module.css";
import "./styles/tokens.css";
import "./styles/base.css";
import type {
  PageDetailResponse,
  PageLayoutJson,
  PageSummary,
  StatusResponse,
} from "./types";

let status: StatusResponse | null = $state(null),
  pages: PageSummary[] = $state([]);
let totalPages = $state(0),
  pagesLoadedThrough = $state(-1),
  pagesLoading = $state(false),
  pagesLoadingMore = $state(false);
let pagesError: ErrorPresentation | null = $state(null),
  loadMoreLocked = false,
  documentGeneration = 0,
  pagesRequest = 0;
let currentPage: number | null = $state(null),
  pageDetail: PageDetailResponse | null = $state(null),
  pageLayout: PageLayoutJson | null = $state(null),
  pageRequest = 0;
let selectedBlockId: number | null = $state(null),
  editorAnchor: EditorAnchor | null = $state(null),
  editorRefreshKey = $state(0);
let importOpen = $state(false),
  importLocked = $state(false),
  searchOpen = $state(false),
  markdownOpen = $state(false),
  unsavedOpen = $state(false),
  pageToolsVisible = $state(false);
let statusMessage: { text: string; kind: "ok" | "error" } | null = $state(null);
let statusMessageTimer: number | undefined;
let contextMenu: { x: number; y: number; page: number; block?: number } | null = $state(null);

function showStatus(text: string, kind: "ok" | "error" = "ok") {
  statusMessage = { text, kind };
  window.clearTimeout(statusMessageTimer);
  statusMessageTimer = window.setTimeout(() => { statusMessage = null; }, 5000);
}

async function copyContextContent() {
  if (!contextMenu) return;
  const target = contextMenu;
  contextMenu = null;
  try {
    const content = target.block === undefined
      ? await api.page.content(target.page)
      : await api.block.content(target.page, target.block);
    if (!(await api.clipboard.writeText(content))) throw new Error("Could not write to the clipboard");
    showStatus(target.block === undefined ? "Page content copied" : "Block content copied");
  } catch (error) {
    reportError(error);
  }
}

function mergePages(existing: PageSummary[], incoming: PageSummary[]) {
  return Object.values(
    Object.fromEntries(
      [...existing, ...incoming].map((page) => [page.index, page]),
    ),
  ).sort((a, b) => a.index - b.index);
}
async function refreshStatus() {
  return (status = await api.document.status());
}
function reportError(error: unknown) {
  const presentation = presentCommandError(error);
  if (presentation.code === "not_loaded") {
    importLocked = false;
    importOpen = true;
  }
  if (presentation.code === "block_not_found") {
    selectedBlockId = null;
    editorAnchor = null;
  }
  showStatus(formatErrorText(presentation.message, presentation.hint), "error");
  return presentation;
}
async function loadPagesWindow(from: number, mode: "replace" | "append") {
  if (mode === "append") {
    if (loadMoreLocked) return;
    loadMoreLocked = true;
    pagesLoadingMore = true;
  } else pagesLoading = true;
  pagesError = null;
  const generation = documentGeneration;
  const request = ++pagesRequest;
  try {
    const data = await api.page.list(from);
    if (generation !== documentGeneration || request !== pagesRequest) return;
    totalPages = data.total_pages;
    pages = mode === "replace" ? data.pages : mergePages(pages, data.pages);
    pagesLoadedThrough = data.page_to;
  } catch (error) {
    if (generation === documentGeneration && request === pagesRequest) {
      pagesError = presentCommandError(error);
    }
  } finally {
    if (generation === documentGeneration && request === pagesRequest) {
      if (mode === "append") pagesLoadingMore = false;
      else pagesLoading = false;
    }
    if (mode === "append" && generation === documentGeneration) {
      loadMoreLocked = false;
    }
  }
}
function loadMorePages() {
  if (
    !pagesLoading &&
    !pagesLoadingMore &&
    pagesLoadedThrough >= 0 &&
    pagesLoadedThrough + 1 < totalPages
  ) {
    void loadPagesWindow(pagesLoadedThrough + 1, "append");
  }
}
async function openPage(
  page: number,
  blockId?: number,
  anchor?: EditorAnchor,
) {
  const request = ++pageRequest;
  try {
    const data = await api.page.view(page);
    if (request !== pageRequest) return;
    currentPage = page;
    pageDetail = data.detail;
    pageLayout = data.layout;
    selectedBlockId = blockId ?? null;
    editorAnchor = blockId === undefined ? null : anchor ?? viewportCenterAnchor();
    if (blockId !== undefined) editorRefreshKey += 1;
  } catch (error) {
    if (request === pageRequest) reportError(error);
  }
}
function resetDocumentView() {
  documentGeneration += 1;
  pagesRequest += 1;
  pageRequest += 1;
  loadMoreLocked = false;
  pagesLoading = false;
  pagesLoadingMore = false;
  currentPage = null;
  pageDetail = null;
  pageLayout = null;
  selectedBlockId = null;
  editorAnchor = null;
  pages = [];
  pagesLoadedThrough = -1;
  totalPages = 0;
}
async function indexed() {
  await refreshStatus();
  resetDocumentView();
  await loadPagesWindow(0, "replace");
  if (totalPages > 0) await openPage(0);
}
async function exportDocument() {
  try {
    const name = await api.export.json();
    if (name) { status = await api.document.status(); showStatus(`Exported ${name}`); }
  } catch (error) { reportError(error); }
}
async function pageUpdated(page: number, blockId?: number) {
  await Promise.all([
    openPage(page, blockId, editorAnchor ?? undefined),
    refreshStatus(),
  ]);
}
async function pageDeleted(page: number, message: string) {
  const nextPage = Math.min(page, totalPages - 2);
  resetDocumentView();
  await Promise.all([loadPagesWindow(0, "replace"), refreshStatus()]);
  await openPage(nextPage);
  showStatus(message);
}
async function moveBlock(page: number, block: number, bbox: string) {
  try {
    const response = await api.block.move(page, block, bbox);
    showStatus(response.message);
    await Promise.all([openPage(page), refreshStatus()]);
    selectedBlockId = block;
    editorAnchor = null;
  } catch (error) {
    reportError(error);
    throw error;
  }
}
function selectBlock(blockId: number, anchor: EditorAnchor) {
  selectedBlockId = blockId;
  editorAnchor = anchor;
  editorRefreshKey += 1;
}
function closeEditor() {
  selectedBlockId = null;
  editorAnchor = null;
}

onMount(() => {
  const restrictContextMenu = (event: MouseEvent) => {
    const target = event.target as HTMLElement | null;
    const copyTarget = target?.closest<HTMLElement>("[data-copy-page]");
    if (copyTarget) {
      event.preventDefault();
      contextMenu = {
        x: Math.min(event.clientX, window.innerWidth - 190),
        y: Math.min(event.clientY, window.innerHeight - 44),
        page: Number(copyTarget.dataset.copyPage),
        ...(copyTarget.dataset.copyBlock === undefined ? {} : { block: Number(copyTarget.dataset.copyBlock) }),
      };
    } else if (!target?.closest("input:not([type='range']):not([type='checkbox']), textarea, [contenteditable='true']")) {
      event.preventDefault();
      contextMenu = null;
    }
  };
  const closeContextMenu = () => { contextMenu = null; };
  window.addEventListener("contextmenu", restrictContextMenu);
  window.addEventListener("pointerdown", closeContextMenu);
  const preventBrowserZoom = (event: WheelEvent) => {
    if (event.ctrlKey || event.metaKey) event.preventDefault();
  };
  window.addEventListener("wheel", preventBrowserZoom, { passive: false });
  const menu = (command: string) => {
    if (command === "import") {
      importLocked = false;
      importOpen = true;
    } else if (command === "export-json") void exportDocument();
    else if (command === "export-markdown") markdownOpen = true;
    else if (command === "search") searchOpen = true;
    else if (command === "page-tools") {
      pageToolsVisible = !pageToolsVisible;
      void api.app.pageToolsVisible(pageToolsVisible);
    }
  };
  const offMenu = api.app.onMenuCommand(menu);
  const offClose = api.app.onCloseRequested(() => { unsavedOpen = true; });
  void api.app.pageToolsVisible(false);
  void (async () => {
    const next = await refreshStatus();
    if (!next.loaded) {
      importOpen = true;
      importLocked = false;
    } else await loadPagesWindow(0, "replace");
  })();
  return () => {
    window.clearTimeout(statusMessageTimer);
    window.removeEventListener("contextmenu", restrictContextMenu);
    window.removeEventListener("pointerdown", closeContextMenu);
    window.removeEventListener("wheel", preventBrowserZoom);
    offMenu();
    offClose();
  };
});
</script>

<svelte:head>
  <title>Paddle JSON Editor</title>
</svelte:head>
<div class={appStyles.shell}>
  <main class={`${appStyles.workspace} ${pageToolsVisible ? appStyles.workspaceTools : ""}`}>
    <aside class={appStyles.panel}>
      <div class={appStyles.head}>
        <h2>Pages</h2>
        <span class={appStyles.hint}>{pages.length} / {totalPages}</span>
      </div>
      <PageList
        {pages}
        {totalPages}
        loading={pagesLoading}
        loadingMore={pagesLoadingMore}
        error={pagesError}
        {currentPage}
        onSelectPage={(page) => page !== currentPage && void openPage(page)}
        onLoadMore={loadMorePages}
        onImport={() => {
          importLocked = false;
          importOpen = true;
        }}
        onRetry={() =>
          void loadPagesWindow(
            pagesLoadedThrough < 0 ? 0 : pagesLoadedThrough + 1,
            pages.length ? "append" : "replace",
          )}
      />
    </aside>
    <section class={appStyles.panel}>
      <PagePanel
        detail={pageDetail}
        layout={pageLayout}
        {selectedBlockId}
        onSelectBlock={selectBlock}
        onMoveBlock={moveBlock}
      />
    </section>
    {#if pageToolsVisible}
      <aside class={appStyles.panel}>
        <div class={appStyles.head}><h2>Page tools</h2></div>
        <PageTools
          page={currentPage}
          {selectedBlockId}
          {totalPages}
          onPageUpdated={(page, blockId) => void pageUpdated(page, blockId)}
          onPageDeleted={(page, message) => void pageDeleted(page, message)}
          onError={reportError}
          onSuccess={(message) => showStatus(message)}
        />
      </aside>
    {/if}
  </main>
  {#if currentPage !== null && selectedBlockId !== null && editorAnchor}<BlockEditor
      page={currentPage}
      blockId={selectedBlockId}
      anchor={editorAnchor}
      refreshKey={editorRefreshKey}
      onClose={closeEditor}
      onSaved={(page, blockId) => void pageUpdated(page, blockId)}
      onDeleted={(page) => {
        closeEditor();
        void openPage(page);
      }}
      onError={reportError}
      onSuccess={(message) => showStatus(message)}
    />{/if}
  <ImportModal
    bind:open={importOpen}
    locked={importLocked}
    hasDocument={status?.loaded ?? false}
    hasChanges={status?.changed ?? false}
    onClose={() => {
      if (!importLocked) importOpen = false;
    }}
    onIndexed={() => {
      importLocked = false;
      importOpen = false;
      void indexed();
    }}
    onSuccess={(message) => showStatus(message)}
  />
  <SearchModal
    bind:open={searchOpen}
    onClose={() => {
      searchOpen = false;
    }}
    onSelectHit={(page, blockId) => void openPage(page, blockId, viewportCenterAnchor())}
    onImport={() => {
      importLocked = false;
      importOpen = true;
    }}
  />
  <UnsavedDialog bind:open={unsavedOpen} onCancel={() => { unsavedOpen = false; }} onDiscard={() => void api.app.confirmClose()} onExport={() => void (async () => { await exportDocument(); if (!(await api.document.status()).changed) await api.app.confirmClose(); })()} />
  <MarkdownExportModal bind:open={markdownOpen} {currentPage} {totalPages} onClose={() => { markdownOpen = false; }} onSuccess={(message) => showStatus(message)} onError={reportError} />
  {#if contextMenu}<div
      class={appStyles.contextMenu}
      role="menu"
      tabindex="-1"
      style:left={`${contextMenu.x}px`}
      style:top={`${contextMenu.y}px`}
      onpointerdown={(event) => event.stopPropagation()}
    ><button type="button" role="menuitem" onclick={() => void copyContextContent()}>{contextMenu.block === undefined ? "Copy Page Content" : "Copy Block Content"}</button></div>{/if}
  <footer class={appStyles.statusBar}>
    <span class={`${appStyles.statusMessage} ${statusMessage?.kind === "error" ? appStyles.statusError : ""}`} aria-live="polite">{statusMessage?.text ?? ""}</span>
    <span>Pages <strong>{status?.total_pages ?? 0}</strong></span><span>Blocks <strong>{status?.total_blocks ?? 0}</strong></span><span>Source <strong>{status?.source ?? "No source"}</strong></span>
  </footer>
</div>
