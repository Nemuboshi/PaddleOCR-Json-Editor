<script lang="ts">
import { tick, type Snippet } from "svelte";
import Button from "./Button.svelte";
import styles from "./dialog.module.css";

let {
  open = $bindable(),
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  destructive = false,
  onconfirm,
}: {
  open: boolean;
  title: string;
  description: string | Snippet;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
  onconfirm: () => void;
} = $props();
let dialog: HTMLDialogElement;

function close() {
  open = false;
}

$effect(() => {
  if (open && !dialog.open) {
    dialog.showModal();
    void tick().then(() => dialog.querySelector<HTMLButtonElement>("button")?.focus());
  } else if (!open && dialog.open) dialog.close();
});
</script>

<dialog
  bind:this={dialog}
  class={`${styles.popup} ${styles.confirm}`}
  aria-label={title}
  oncancel={(event) => {
    event.preventDefault();
    close();
  }}
  onclick={(event) => {
    if (event.target === dialog) close();
  }}
  onclose={close}
>
  <header class={styles.header}>
    <div>
      <h2 class={styles.title}>{title}</h2>
      <div class={styles.subtitle}>
        {#if typeof description === "string"}{description}{:else}{@render description()}{/if}
      </div>
    </div>
  </header>
  <div class={styles.body}>
    <div class={styles.actions}>
      <Button type="button" variant="ghost" onclick={close}>{cancelLabel}</Button>
      <Button type="button" variant={destructive ? "danger" : "primary"} onclick={() => { close(); onconfirm(); }}>{confirmLabel}</Button>
    </div>
  </div>
</dialog>
