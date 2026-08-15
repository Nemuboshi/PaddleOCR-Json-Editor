<script lang="ts">
import type { ErrorPresentation } from "../../lib/errors";
import appStyles from "../../styles/app.module.css";
import stateStyles from "../../styles/states.module.css";
import type { PageSummary } from "../../types";
import Button from "../../ui/Button.svelte";
import ErrorState from "../ErrorState.svelte";
import styles from "./PageList.module.css";

let {
  pages,
  totalPages,
  loading,
  loadingMore,
  error,
  currentPage,
  onSelectPage,
  onLoadMore,
  onImport,
  onRetry,
}: {
  pages: PageSummary[];
  totalPages: number;
  loading: boolean;
  loadingMore: boolean;
  error: ErrorPresentation | null;
  currentPage: number | null;
  onSelectPage: (page: number) => void;
  onLoadMore: () => void;
  onImport?: () => void;
  onRetry?: () => void;
} = $props();

let hasMore = $derived(pages.length < totalPages);

function loadNearBottom(event: Event) {
  const element = event.currentTarget as HTMLDivElement;
  if (
    hasMore &&
    !loading &&
    !loadingMore &&
    element.scrollTop + element.clientHeight >= element.scrollHeight - 320
  ) {
    onLoadMore();
  }
}

function moveSelection(event: KeyboardEvent) {
  if (event.key !== "ArrowUp" && event.key !== "ArrowDown") return;
  event.preventDefault();
  const row = (event.currentTarget as HTMLElement).parentElement;
  const next = event.key === "ArrowUp"
    ? row?.previousElementSibling
    : row?.nextElementSibling;
  const button = next?.querySelector<HTMLButtonElement>("button[data-page]");
  if (!button) return;
  button.focus();
  button.scrollIntoView({ block: "nearest" });
  onSelectPage(Number(button.dataset.page));
}
</script>

{#if loading && pages.length === 0}
  <div class={appStyles.body}>
  <div class={stateStyles.loading}>Loading pages…</div>
</div>
{:else if error && pages.length === 0}
  <div class={appStyles.body}>
  <ErrorState presentation={error} {onImport} {onRetry} />
</div>
{:else if !loading && pages.length === 0}
  <div class={appStyles.body}>
  <div class={`${stateStyles.empty} ${stateStyles.compact}`}>
      <p>No pages loaded.</p>
      {#if onImport}<Button type="button" variant="primary" size="sm" onclick={onImport}
          >Import JSON</Button
        >{/if}
    </div>
</div>
{:else}
  <div class={`${appStyles.body} ${styles.body}`}>
  {#if error}<div class={`${styles.meta} muted`}><Button
        type="button"
        size="sm"
        onclick={onRetry}>Retry</Button
      ></div>{/if}
  <div class={styles.scroll} onscroll={loadNearBottom}>
      {#each pages as page (page.index)}
        <div class={styles.row}>
          <Button
            type="button"
            variant="bare"
            class={`${styles.item} ${currentPage === page.index ? styles.itemActive : ""}`}
            data-page={page.index}
            data-copy-page={page.index}
            onclick={() => onSelectPage(page.index)}
            onkeydown={moveSelection}
          >
            <span class={styles.num}>p{page.index}</span><span class={styles.metaText}
              >{page.block_count} blocks</span
            >
          </Button>
        </div>
      {/each}
      {#if loadingMore}<div class={`${styles.footer} muted`}>Loading more…</div>{/if}
      {#if !hasMore && pages.length}<div class={`${styles.footer} muted`}>
          All pages loaded
        </div>{/if}
    </div>
</div>
{/if}
