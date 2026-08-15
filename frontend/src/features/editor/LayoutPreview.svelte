<script lang="ts">
import { onMount, tick } from "svelte";
import { anchorFromRect, type EditorAnchor } from "../../lib/anchor";
import { LABEL_COLORS } from "../../lib/utils";
import type { PageLayoutJson } from "../../types";
import Button from "../../ui/Button.svelte";
import styles from "./LayoutPreview.module.css";

let {
  page,
  layout,
  zoom,
  boxed,
  selectedBlockId,
  onSelectBlock,
  onMoveBlock,
  onZoom,
}: {
  page: number;
  layout: PageLayoutJson;
  zoom: number;
  boxed: boolean;
  selectedBlockId: number | null;
  onSelectBlock: (blockId: number, anchor: EditorAnchor) => void;
  onMoveBlock: (page: number, block: number, bbox: string) => Promise<void>;
  onZoom: (zoom: number) => void;
} = $props();
let stage: HTMLDivElement | null = $state(null);
let shell: HTMLDivElement | null = $state(null);
let pageElement: HTMLDivElement | null = $state(null);
let pageWidth = $state(0), pageHeight = $state(0);
let backgroundReady = $state(false),
  backgroundFailed = $state(false),
  backgroundRetries = $state(0);
let scale = $state(1), panX = $state(0), panY = $state(0);
let movedBBox: { block: number; bbox: [number, number, number, number] } | null = $state(null);
let suppressClick = false;
type DragState = {
  mode: "pan" | "block";
  pointer: number;
  startX: number;
  startY: number;
  originX: number;
  originY: number;
  block?: number;
  bbox?: [number, number, number, number];
  moved: boolean;
};
let drag = $state<DragState | null>(null);

function updateScale() {
  if (!stage || !shell || !pageElement) return;
  const container = stage.parentElement ?? stage;
  const style = getComputedStyle(container);
  const availableWidth = Math.max(
    container.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight) - 16,
    40,
  );
  const availableHeight = Math.max(
    container.clientHeight - parseFloat(style.paddingTop) - parseFloat(style.paddingBottom) - 16,
    40,
  );
  scale = Math.min(
    1,
    availableWidth / pageWidth,
    availableHeight / pageHeight,
  ) * zoom / 100;
  pageElement.style.transformOrigin = "top left";
  pageElement.style.transform = `scale(${scale})`;
  shell.style.width = `${pageWidth * scale}px`;
  shell.style.height = `${pageHeight * scale}px`;
  shell.style.transform = `translate(${panX}px, ${panY}px)`;
  shell.style.overflow = "hidden";
}

$effect(() => {
  const currentBackground = backgroundImage;
  pageWidth = 0;
  pageHeight = 0;
  backgroundReady = false;
  backgroundFailed = false;
  backgroundRetries = 0;
  movedBBox = null;
  void currentBackground;
});

function backgroundLoaded(event: Event) {
  const image = event.currentTarget as HTMLImageElement;
  if (!image.naturalWidth || !image.naturalHeight) return;
  pageWidth = image.naturalWidth;
  pageHeight = image.naturalHeight;
  backgroundReady = true;
  backgroundFailed = false;
  void tick().then(updateScale);
}

function backgroundErrored() {
  if (backgroundRetries === 0) {
    backgroundRetries = 1;
    return;
  }
  backgroundFailed = true;
}

function dragStart(event: PointerEvent) {
  if (event.button !== 0 || !backgroundReady) return;
  const blockElement = (event.target as Element).closest<HTMLElement>("[data-block-id]");
  const moveBlock = !!blockElement && (event.ctrlKey || event.metaKey);
  if (blockElement && !moveBlock) return;
  const block = moveBlock
    ? layout.blocks.find((item) => item.block_id === Number(blockElement.dataset.blockId))
    : undefined;
  event.preventDefault();
  stage?.focus();
  stage?.setPointerCapture(event.pointerId);
  drag = {
    mode: block ? "block" : "pan",
    pointer: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    originX: panX,
    originY: panY,
    ...(block ? { block: block.block_id, bbox: [...block.bbox] as [number, number, number, number] } : {}),
    moved: false,
  };
}

function dragMove(event: PointerEvent) {
  if (!drag || drag.pointer !== event.pointerId) return;
  const dx = event.clientX - drag.startX;
  const dy = event.clientY - drag.startY;
  drag.moved ||= Math.abs(dx) + Math.abs(dy) > 3;
  if (drag.mode === "pan") {
    panX = drag.originX + dx;
    panY = drag.originY + dy;
    updateScale();
    return;
  }
  const [x1, y1, x2, y2] = drag.bbox!;
  const width = x2 - x1;
  const height = y2 - y1;
  const nextX = Math.max(0, Math.min(pageWidth - width, x1 + dx / scale));
  const nextY = Math.max(0, Math.min(pageHeight - height, y1 + dy / scale));
  movedBBox = {
    block: drag.block!,
    bbox: [Math.round(nextX), Math.round(nextY), Math.round(nextX + width), Math.round(nextY + height)],
  };
}

function dragEnd(event: PointerEvent) {
  if (!drag || drag.pointer !== event.pointerId) return;
  const finished = drag;
  drag = null;
  suppressClick = finished.moved;
  if (suppressClick) setTimeout(() => { suppressClick = false; });
  if (finished.mode === "block" && finished.moved && movedBBox) {
    const moved = movedBBox;
    void onMoveBlock(page, moved.block, moved.bbox.join(",")).catch(() => { movedBBox = null; });
  }
}

function dragCancel(event: PointerEvent) {
  if (drag?.pointer !== event.pointerId) return;
  if (drag.mode === "block") movedBBox = null;
  drag = null;
}

$effect(() => {
  const dimensions = { pageWidth, pageHeight, backgroundReady };
  if (dimensions.backgroundReady && dimensions.pageWidth && dimensions.pageHeight) updateScale();
});
onMount(() => {
  if (!stage || typeof ResizeObserver === "undefined") {
    window.addEventListener("resize", updateScale);
    return () => window.removeEventListener("resize", updateScale);
  }
  const observer = new ResizeObserver(updateScale);
  observer.observe(stage.parentElement ?? stage);
  return () => observer.disconnect();
});

let backgroundImage = $derived(boxed && layout.boxed_image ? layout.boxed_image : layout.input_image);
let backgroundSrc = $derived(
  backgroundImage && backgroundRetries > 0 && backgroundImage.startsWith("/session-assets/")
    ? `${backgroundImage}?retry=${backgroundRetries}`
    : (backgroundImage ?? ""),
);
let showOverlay = $derived(backgroundReady && !backgroundFailed);
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  class={`${styles.stage} ${drag?.mode === "pan" ? styles.panning : ""}`}
  bind:this={stage}
  data-page={page}
  role="application"
  tabindex="0"
  aria-label="Page canvas"
  onwheel={(event) => {
    if (!stage?.contains(document.activeElement)) return;
    event.preventDefault();
    onZoom(zoom + (event.deltaY < 0 ? 10 : -10));
  }}
  onpointerdown={dragStart}
  onpointermove={dragMove}
  onpointerup={dragEnd}
  onpointercancel={dragCancel}
>
  {#if !backgroundImage || backgroundFailed}<div class={styles.message}>The page image could not be displayed.</div>
  {:else}
    {#if !backgroundReady}<div class={styles.message}>Loading image…</div>{/if}
    <div class={`${styles.shell} ${!backgroundReady ? styles.loading : ""}`} bind:this={shell}>
      <div
        class={styles.page}
        bind:this={pageElement}
        style:width={`${pageWidth}px`}
        style:height={`${pageHeight}px`}
      >
          {#key backgroundSrc}<img
            class={styles.background}
            src={backgroundSrc}
            alt="page"
            referrerpolicy="no-referrer"
            draggable="false"
            onload={backgroundLoaded}
            onerror={backgroundErrored}
            style:width={`${pageWidth}px`}
            style:height={`${pageHeight}px`}
          />{/key}
          {#if showOverlay}<div
            class={styles.overlay}
            style:width={`${pageWidth}px`}
            style:height={`${pageHeight}px`}
          >
            {#each layout.blocks as block (block.block_id)}
              {@const bbox = movedBBox?.block === block.block_id ? movedBBox.bbox : block.bbox}
              {@const x1 = bbox[0]}{@const y1 = bbox[1]}{@const width = Math.max(
                bbox[2] - x1,
                1,
              )}{@const height = Math.max(bbox[3] - y1, 1)}
              <Button
                type="button"
                variant="bare"
                class={`${styles.block} ${drag?.mode === "block" && drag.block === block.block_id ? styles.blockDragging : ""}`}
                data-block-id={block.block_id}
                data-active={block.block_id === selectedBlockId ? "true" : undefined}
                data-copy-page={page}
                data-copy-block={block.block_id}
                style={`left:${x1}px;top:${y1}px;width:${width}px;height:${height}px;--block-color:${LABEL_COLORS[block.label] ?? "#adb5bd"}`}
                onclick={(event: MouseEvent) => {
                  if (suppressClick) {
                    suppressClick = false;
                    return;
                  }
                  onSelectBlock(
                    block.block_id,
                    anchorFromRect(
                      (event.currentTarget as HTMLElement).getBoundingClientRect(),
                      "right",
                    ),
                  );
                }}
              >
                <span class={styles.tag}>#{block.block_id} {block.label}</span>
              </Button>
            {/each}
          </div>{/if}
        </div>
    </div>
  {/if}
</div>
