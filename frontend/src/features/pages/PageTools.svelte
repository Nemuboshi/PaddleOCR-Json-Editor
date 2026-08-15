<script lang="ts">
import { api } from "../../api";
import appStyles from "../../styles/app.module.css";
import stateStyles from "../../styles/states.module.css";
import type { MessageResponse } from "../../types";
import Button from "../../ui/Button.svelte";
import ConfirmDialog from "../../ui/ConfirmDialog.svelte";
import Field from "../../ui/Field.svelte";
import fieldStyles from "../../ui/field.module.css";
import styles from "./PageTools.module.css";

let {
  page,
  selectedBlockId,
  totalPages,
  onPageUpdated,
  onPageDeleted,
  onError,
  onSuccess,
}: {
  page: number | null;
  selectedBlockId: number | null;
  totalPages: number;
  onPageUpdated: (page: number, blockId?: number) => void;
  onPageDeleted: (page: number, message: string) => void;
  onError: (error: unknown) => void;
  onSuccess: (message: string) => void;
} = $props();
let busy = $state(false), downloading = $state(false), deleteOpen = $state(false);

async function downloadAssets() {
  busy = true;
  downloading = true;
  try {
    const result = await api.task.downloadAssets();
    onSuccess(`Downloaded ${result.downloaded} images. ${result.failed} failed.`);
    if (result.downloaded > 0 && page !== null) {
      onPageUpdated(page, selectedBlockId ?? undefined);
    }
  } catch (error) { onError(error); } finally { busy = false; downloading = false; }
}

async function deletePage() {
  if (page === null) return;
  busy = true;
  try {
    const response = await api.page.delete(page);
    deleteOpen = false;
    onPageDeleted(page, response.message);
  } catch (error) {
    onError(error);
  } finally {
    busy = false;
  }
}

async function runTool(
  tool: "merge" | "split" | "reorder",
  form: HTMLFormElement,
) {
  if (page === null) return;
  const data = new FormData(form);
  busy = true;
  try {
    let response: MessageResponse;
    if (tool === "merge") {
      response = await api.block.merge(page, String(data.get("blocks") ?? ""));
    } else if (tool === "split") {
      response = await api.block.split(
        page,
        Number(data.get("block")),
        Number(data.get("at")),
      );
    } else {response = await api.block.reorder(
        page,
        String(data.get("blocks") ?? ""),
      );}
    onSuccess(response.message);
    onPageUpdated(page, selectedBlockId ?? undefined);
  } catch (error) {
    onError(error);
  } finally {
    busy = false;
  }
}
</script>

{#if page === null}
  <div class={appStyles.body}>
  <div class={`${stateStyles.empty} ${stateStyles.compact}`}>
    <p>Open a page to use merge / split / reorder.</p>
  </div>
</div>
{:else}
  <div class={`${appStyles.body} ${styles.body}`}>
  <div class={styles.panel}>
    <form
      class={styles.form}
      onsubmit={(event) => {
          event.preventDefault();
          void runTool("merge", event.currentTarget);
        }}
    >
      <Field label="Merge blocks"
      ><input
            class={fieldStyles.input}
            name="blocks"
            placeholder="1,2,3"
            disabled={busy}
          /></Field>
      <Button type="submit" size="sm" disabled={busy}>Merge</Button>
    </form>
    <form
      class={styles.form}
      onsubmit={(event) => {
          event.preventDefault();
          void runTool("split", event.currentTarget);
        }}
    >
      <Field label="Split block id"
      ><input
            class={fieldStyles.input}
            type="number"
            name="block"
            placeholder="id"
            disabled={busy}
          /></Field>
      <Field label="At char index"
      ><input
            class={fieldStyles.input}
            type="number"
            name="at"
            placeholder="0"
            disabled={busy}
          /></Field>
      <Button type="submit" size="sm" disabled={busy}>Split</Button>
    </form>
    <form
      class={styles.form}
      onsubmit={(event) => {
          event.preventDefault();
          void runTool("reorder", event.currentTarget);
        }}
    >
      <Field label="New order"
      ><input
            class={fieldStyles.input}
            name="blocks"
            placeholder="2,0,1,3"
            disabled={busy}
          /></Field>
      <Button type="submit" size="sm"
        disabled={busy}>Reorder</Button>
    </form>
  </div>
  <div class={styles.footer}>
    {#if downloading}<Button type="button" size="sm" onclick={() => void api.task.cancel()}>Cancel task</Button>
    {:else}<Button type="button" size="sm" disabled={busy} onclick={() => void downloadAssets()}>Download assets</Button>{/if}
    <Button type="button" variant="danger" size="sm" disabled={busy || totalPages <= 1} title={totalPages <= 1 ? "The document must keep at least one page" : undefined} onclick={() => { deleteOpen = true; }}>Delete page</Button>
  </div>
</div>
<ConfirmDialog bind:open={deleteOpen} title={`Delete page ${page}?`} description="Blocks and page-specific assets will be removed. This cannot be undone." confirmLabel="Delete page" destructive onconfirm={() => void deletePage()} />
{/if}
