<script lang="ts">
import Button from "../../ui/Button.svelte";
import Dialog from "../../ui/Dialog.svelte";
import dialogStyles from "../../ui/dialog.module.css";

let { open = $bindable(), onExport, onDiscard, onCancel }: {
  open: boolean; onExport: () => void; onDiscard: () => void; onCancel: () => void;
} = $props();
</script>
<Dialog bind:open locked title="Export changes?" subtitle="The application does not restore temporary edits after exit." onclose={onCancel}>
  <div class={dialogStyles.actions}>
    <Button type="button" variant="ghost" onclick={onCancel}>Cancel</Button>
    <Button type="button" onclick={onDiscard}>Discard</Button>
    <Button type="button" variant="primary" onclick={onExport}>Export</Button>
  </div>
</Dialog>
