<script lang="ts">
import { onMount } from "svelte";
import { api, type TaskProgress } from "../../api";
import { formatErrorText } from "../../copy";
import { presentCommandError } from "../../lib/errors";
import { compactFileName } from "../../lib/utils";
import Button from "../../ui/Button.svelte";
import Checkbox from "../../ui/Checkbox.svelte";
import Dialog from "../../ui/Dialog.svelte";
import dialogStyles from "../../ui/dialog.module.css";
import styles from "./ImportModal.module.css";
import UnsavedDialog from "./UnsavedDialog.svelte";

let { open = $bindable(), locked, hasDocument, hasChanges, onClose, onIndexed, onSuccess }: {
  open: boolean; locked: boolean; hasDocument: boolean; hasChanges: boolean; onClose: () => void;
  onIndexed: () => void; onSuccess: (message: string) => void;
} = $props();
let status: string | null = $state(null), statusError = $state(false), submitting = $state(false);
let selectedPath = $state(""), downloadAssets = $state(false), confirmReplaceOpen = $state(false);
let progress: TaskProgress | null = $state(null);

onMount(() =>
  api.task.onProgress((event) => {
    if (event.task === "import") progress = event;
  }),
);

async function chooseFile() {
  const path = await api.document.selectImportFile();
  if (path) selectedPath = path;
}
function importSelected() {
  if (hasDocument && hasChanges) confirmReplaceOpen = true;
  else void runImport();
}
async function runImport() {
  if (!selectedPath) return;
  submitting = true; statusError = false; progress = null;
  status = downloadAssets ? "Importing and downloading assets…" : "Importing JSON…";
  try {
    const response = await api.document.import(selectedPath, downloadAssets);
    onIndexed();
    onSuccess(`Indexed ${response.total_pages} pages (${response.total_blocks} blocks)`);
    status = null;
    selectedPath = "";
  } catch (cause) {
    const error = presentCommandError(cause);
    status = formatErrorText(error.message, error.hint); statusError = true;
  } finally { submitting = false; }
}
</script>

<Dialog bind:open locked={locked || submitting} title="Import & index JSON" initialFocus="#select-import-file" onclose={onClose}>
  <div id="import-form" class={styles.form}>
    <div class={styles.fileRow}>
      <Button id="select-import-file" type="button" disabled={submitting} onclick={() => void chooseFile()}>
        Select file
      </Button>
      <span class={styles.fileName} title={selectedPath}>
        {selectedPath ? compactFileName(selectedPath) : "No file selected"}
      </span>
    </div>
    <Checkbox id="download-assets" bind:checked={downloadAssets}>
      {#snippet label()}Download remote images{/snippet}
    </Checkbox>
    {#if status}<div class={`${styles.status} ${statusError ? styles.statusError : ""}`}>
        <span>{status}</span>
        {#if submitting && progress}
          {#if progress.total}<progress max={progress.total} value={progress.done}></progress>{:else}<progress></progress>{/if}
          <span>{progress.stage}{progress.total ? ` (${progress.done} / ${progress.total})` : ""}</span>
        {/if}
      </div>{/if}
    <div class={dialogStyles.actions}>
      {#if !locked && !submitting}<Button type="button" variant="ghost" onclick={onClose}>Cancel</Button>{/if}
      {#if submitting}<Button type="button" onclick={() => void api.task.cancel()}>Cancel task</Button>{/if}
      <Button type="button" variant="primary" disabled={submitting || !selectedPath} onclick={importSelected}>Import</Button>
    </div>
  </div>
</Dialog>

<UnsavedDialog
  bind:open={confirmReplaceOpen}
  onCancel={() => { confirmReplaceOpen = false; }}
  onDiscard={() => { confirmReplaceOpen = false; void runImport(); }}
  onExport={() => void (async () => {
    const name = await api.export.json();
    if (name) { onSuccess(`Exported ${name}`); confirmReplaceOpen = false; void runImport(); }
  })()}
/>
