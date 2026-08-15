<script lang="ts">
import { api } from "../../api";
import Button from "../../ui/Button.svelte";
import Dialog from "../../ui/Dialog.svelte";
import Field from "../../ui/Field.svelte";
import dialogStyles from "../../ui/dialog.module.css";
import fieldStyles from "../../ui/field.module.css";

let { open = $bindable(), currentPage, totalPages, onClose, onSuccess, onError }: {
  open: boolean; currentPage: number | null; totalPages: number;
  onClose: () => void; onSuccess: (message: string) => void; onError: (error: unknown) => void;
} = $props();
let start = $state(1), end = $state(1), busy = $state(false), wasOpen = false;
$effect(() => {
  if (open && !wasOpen) {
    start = (currentPage ?? 0) + 1;
    end = start;
  }
  wasOpen = open;
});
async function submit() {
  busy = true;
  try {
    const name = await api.export.markdown(start, end);
    if (name) { onSuccess(`Exported ${name}`); onClose(); }
  } catch (error) { onError(error); } finally { busy = false; }
}
</script>
<Dialog bind:open locked={busy} title="Export Markdown" subtitle="Select an inclusive page range." initialFocus="#markdown-start" onclose={onClose}>
  <div class={fieldStyles.grid}>
    <Field label="Start page"><input class={fieldStyles.input} id="markdown-start" type="number" min="1" max={totalPages} disabled={busy} bind:value={start} /></Field>
    <Field label="End page"><input class={fieldStyles.input} type="number" min="1" max={totalPages} disabled={busy} bind:value={end} /></Field>
  </div>
  <div class={dialogStyles.actions}>
    <Button type="button" variant="ghost" disabled={busy} onclick={() => { start = 1; end = totalPages; }}>All pages</Button>
    <Button type="button" variant="ghost" disabled={busy} onclick={onClose}>Cancel</Button>
    <Button type="button" variant="primary" disabled={busy || !Number.isInteger(start) || !Number.isInteger(end) || start < 1 || end < start || end > totalPages} onclick={() => void submit()}>{busy ? "Exporting…" : "Export"}</Button>
  </div>
</Dialog>
