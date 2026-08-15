<script lang="ts">
import { onMount } from "svelte";
import { api } from "../../api";
import type { EditorAnchor } from "../../lib/anchor";
import { presentCommandError } from "../../lib/errors";
import stateStyles from "../../styles/states.module.css";
import type { BlockDetailResponse } from "../../types";
import Button from "../../ui/Button.svelte";
import ConfirmDialog from "../../ui/ConfirmDialog.svelte";
import dialogStyles from "../../ui/dialog.module.css";
import Field from "../../ui/Field.svelte";
import fieldStyles from "../../ui/field.module.css";
import styles from "./BlockEditor.module.css";

const PANEL_WIDTH = 360,
  VIEWPORT_GAP = 8;
let {
  page,
  blockId,
  anchor,
  refreshKey,
  onClose,
  onSaved,
  onDeleted,
  onError,
  onSuccess,
}: {
  page: number;
  blockId: number;
  anchor: EditorAnchor;
  refreshKey: number;
  onClose: () => void;
  onSaved: (page: number, blockId: number) => void;
  onDeleted: (page: number) => void;
  onError: (error: unknown) => void;
  onSuccess: (message: string) => void;
} = $props();
let panel: HTMLElement | null = $state(null),
  data: BlockDetailResponse | null = $state(null),
  loading = $state(false);
let label = $state(""),
  content = $state(""),
  bbox = $state(""),
  order = $state(""),
  bboxHint: string | null = $state(null);
let saving = $state(false),
  deleteOpen = $state(false),
  position = $state({ left: VIEWPORT_GAP, top: VIEWPORT_GAP });
let drag: {
  pointerId: number;
  startX: number;
  startY: number;
  originLeft: number;
  originTop: number;
} | null = null;

function clamp(
  x: number,
  y: number,
  width = Math.min(PANEL_WIDTH, window.innerWidth - 16),
  height = 0,
) {
  return {
    left: Math.min(
      Math.max(VIEWPORT_GAP, x),
      window.innerWidth - width - VIEWPORT_GAP,
    ),
    top: Math.min(
      Math.max(VIEWPORT_GAP, y),
      window.innerHeight - height - VIEWPORT_GAP,
    ),
  };
}
function fit(x: number, y: number, flip = false) {
  const rect = panel?.getBoundingClientRect(),
    width = rect?.width ?? Math.min(PANEL_WIDTH, window.innerWidth - 16),
    height = rect?.height ?? 0;
  return clamp(
    flip && anchor.leftEdge !== undefined &&
      anchor.x + width > window.innerWidth
      ? anchor.leftEdge - width
      : x,
    y,
    width,
    height,
  );
}

$effect(() => {
  const { x, y } = anchor;
  position = fit(x, y, true);
});
$effect(() => {
  const load = { page, blockId, refreshKey };
  let cancelled = false;
  loading = true;
  api.block
    .detail(load.page, load.blockId)
    .then((block) => {
      if (cancelled) return;
      data = block;
      label = block.label;
      content = block.content;
      bbox = block.bbox;
      order = block.order;
      bboxHint = null;
    })
    .catch(onError)
    .finally(() => {
      if (!cancelled) loading = false;
    });
  return () => {
    cancelled = true;
  };
});

onMount(() => {
  const refit = () => {
    position = fit(position.left, position.top);
  };
  refit();
  window.addEventListener("resize", refit);
  const observer = typeof ResizeObserver === "undefined"
    ? null
    : new ResizeObserver(refit);
  if (panel) observer?.observe(panel);
  const keydown = (event: KeyboardEvent) => {
    if (event.key === "Escape" && !deleteOpen) onClose();
  };
  const outside = (event: PointerEvent) => {
    const target = event.target as Node | null;
    if (deleteOpen || panel?.contains(target)) return;
    if (target instanceof Element && target.closest("[role='dialog']")) return;
    onClose();
  };
  window.addEventListener("keydown", keydown);
  const timer = window.setTimeout(
    () => window.addEventListener("pointerdown", outside, true),
    0,
  );
  return () => {
    observer?.disconnect();
    window.removeEventListener("resize", refit);
    window.removeEventListener("keydown", keydown);
    window.clearTimeout(timer);
    window.removeEventListener("pointerdown", outside, true);
  };
});

function dragStart(event: PointerEvent) {
  if (
    event.button !== 0 ||
    (event.target as HTMLElement | null)?.closest(
      "button, a, input, textarea, select",
    )
  ) {
    return;
  }
  event.preventDefault();
  (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
  drag = {
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    originLeft: position.left,
    originTop: position.top,
  };
}
function dragMove(event: PointerEvent) {
  if (drag?.pointerId === event.pointerId) {
    position = fit(
      drag.originLeft + event.clientX - drag.startX,
      drag.originTop + event.clientY - drag.startY,
    );
  }
}
function dragEnd(event: PointerEvent) {
  if (drag?.pointerId === event.pointerId) drag = null;
}
async function save(event: SubmitEvent) {
  event.preventDefault();
  saving = true;
  bboxHint = null;
  try {
    const response = await api.block.update(page, blockId, {
      label,
      content,
      bbox,
      order,
    });
    onSuccess(response.message);
    onSaved(page, blockId);
  } catch (cause) {
    const presentation = presentCommandError(cause);
    if (presentation.code === "invalid_bbox") bboxHint = presentation.hint;
    onError(cause);
  } finally {
    saving = false;
  }
}
async function remove() {
  try {
    const response = await api.block.delete(page, blockId);
    onSuccess(response.message);
    onDeleted(page);
  } catch (cause) {
    onError(cause);
  }
}
</script>

<section
  bind:this={panel}
  class={styles.panel}
  style:left={`${position.left}px`}
  style:top={`${position.top}px`}
  style:width={`min(${PANEL_WIDTH}px, calc(100vw - 16px))`}
  aria-label={`Edit block ${blockId}`}
>
  <header
    class={styles.head}
    role="toolbar"
    tabindex="-1"
    aria-label={`Edit block ${blockId}`}
    onpointerdown={dragStart}
    onpointermove={dragMove}
    onpointerup={dragEnd}
    onpointercancel={dragEnd}
  >
    <div>
      <h3>Block #{blockId}</h3>
      <p class={styles.subtle}>Page {page}</p>
    </div>
    <Button
      type="button"
      size="sm"
      variant="ghost"
      class={dialogStyles.close}
      aria-label="Close"
      title="Close"
      onpointerdown={(event: PointerEvent) => event.stopPropagation()}
      onclick={(event: MouseEvent) => {
        event.stopPropagation();
        onClose();
      }}>✕</Button>
  </header>
  <div class={styles.body}>
    {#if loading || !data}<div class={stateStyles.loadingCompact}>Loading block…</div>
    {:else}<div class={styles.editor}>
        <form id="block-form" class={styles.form} onsubmit={save}>
          <Field label="Label"
            ><input class={fieldStyles.input} name="label" bind:value={label} /></Field
          >
          <Field label="Content"
            ><textarea class={fieldStyles.textarea} name="content" rows="8" bind:value={content}
            ></textarea></Field
          >
          <Field label="BBox (x1,y1,x2,y2)"
            ><input
              class={fieldStyles.input}
              name="bbox"
              bind:value={bbox}
              oninput={() => {
                bboxHint = null;
              }}
            />{#if bboxHint}<p class={styles.fieldError}>{bboxHint}</p>{/if}</Field
          >
          <Field label="Order (empty = unchanged)"
            ><input class={fieldStyles.input} name="order" bind:value={order} /></Field
          >
          <div class={styles.actions}>
            <Button type="submit" variant="primary" disabled={saving}>Save changes</Button
            ><Button
              type="button"
              variant="danger"
              onclick={() => {
                deleteOpen = true;
              }}>Delete</Button
            >
          </div>
        </form>
      </div>{/if}
  </div>
  <ConfirmDialog
    bind:open={deleteOpen}
    title={`Delete block ${blockId}?`}
    description="This action cannot be undone."
    confirmLabel="Delete"
    destructive
    onconfirm={() => void remove()}
  />
</section>
