<script lang="ts">
import { anchorFromRect, type EditorAnchor } from "../../lib/anchor";
import { stripHtml } from "../../lib/utils";
import appStyles from "../../styles/app.module.css";
import stateStyles from "../../styles/states.module.css";
import type { PageDetailResponse, PageLayoutJson } from "../../types";
import Button from "../../ui/Button.svelte";
import LayoutPreview from "./LayoutPreview.svelte";
import styles from "./PagePanel.module.css";

let {
  detail,
  layout,
  selectedBlockId,
  onSelectBlock,
  onMoveBlock,
}: {
  detail: PageDetailResponse | null;
  layout: PageLayoutJson | null;
  selectedBlockId: number | null;
  onSelectBlock: (blockId: number, anchor: EditorAnchor) => void;
  onMoveBlock: (page: number, block: number, bbox: string) => Promise<void>;
} = $props();

let zoom = $state(100);
let boxed = $state(false);

function setZoom(value: number) {
  zoom = Math.max(50, Math.min(200, value));
}
</script>

{#if !detail || !layout}
  <div class={`${appStyles.body} ${styles.body}`}>
  <div class={stateStyles.empty}>
    <div class={stateStyles.icon}>📄</div>
    <h2>Select a page</h2>
    <p>Choose a page from the left sidebar to preview blocks and edit OCR layout.</p>
  </div>
</div>
{:else}
  <div class={`${appStyles.body} ${styles.body}`}>
  <div class={styles.view} data-page={detail.page_index}>
    <header class={styles.header}>
      <div>
        <h2>Page {detail.page_index}</h2>
        <p class="muted">{detail.block_count} blocks</p>
      </div>
      <div class={styles.controls}>
        <label class={styles.zoom}>
          <span>Zoom</span>
          <input type="range" min="50" max="200" step="10" bind:value={zoom} />
          <output>{zoom}%</output>
        </label>
        <label class={`${styles.switch} ${!layout.boxed_image ? styles.disabled : ""}`}>
          <span>Original</span>
          <input type="checkbox" role="switch" bind:checked={boxed} disabled={!layout.boxed_image} />
          <span>Boxes</span>
        </label>
      </div>
    </header>
    <div class={styles.layout}>
      <section class={styles.canvas}>
        {#key detail.page_index}<LayoutPreview page={detail.page_index} {layout} {zoom} {boxed}
            {selectedBlockId} {onSelectBlock} {onMoveBlock} onZoom={setZoom} />{/key}
      </section>
      <section class={styles.blocks}>
        <h3>Blocks</h3>
        <div class={styles.scroll}>
          {#each detail.blocks as block (block.block_id)}
            <div class={styles.row}>
              <Button
                type="button"
                variant="bare"
                class={`${styles.item} ${selectedBlockId === block.block_id ? styles.itemActive : ""}`}
                data-copy-page={detail.page_index}
                data-copy-block={block.block_id}
                onclick={(event: MouseEvent) =>
                  onSelectBlock(
                    block.block_id,
                    anchorFromRect(
                      (event.currentTarget as HTMLElement).getBoundingClientRect(),
                      "right",
                    ),
                  )}
              >
                <span class={styles.id}>#{block.block_id}</span><span class={styles.chip}
                  >{block.label}{block.order !== "-"
                    ? ` · o${block.order}`
                    : ""}</span
                ><span class={styles.preview}>{stripHtml(block.preview)}</span>
              </Button>
            </div>
          {/each}
        </div>
      </section>
    </div>
  </div>
</div>
{/if}
