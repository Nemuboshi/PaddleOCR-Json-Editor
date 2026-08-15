<script lang="ts">
import { tick, type Snippet } from "svelte";
import Button from "./Button.svelte";
import styles from "./dialog.module.css";

let {
  open = $bindable(),
  locked = false,
  wide = false,
  title,
  subtitle,
  initialFocus,
  children,
  onclose,
}: {
  open: boolean;
  locked?: boolean;
  wide?: boolean;
  title: string;
  subtitle?: string;
  initialFocus?: string;
  children: Snippet;
  onclose?: () => void;
} = $props();
let dialog: HTMLDialogElement;

function close() {
  if (locked) return;
  open = false;
  onclose?.();
}

$effect(() => {
  if (open && !dialog.open) {
    dialog.showModal();
    void tick().then(() =>
      (initialFocus
        ? dialog.querySelector<HTMLElement>(initialFocus)
        : dialog.querySelector<HTMLElement>("button, input, textarea, select")
      )?.focus(),
    );
  } else if (!open && dialog.open) dialog.close();
});
</script>

<dialog
  bind:this={dialog}
  class={`${styles.popup} ${wide ? styles.wide : ""}`}
  aria-label={title}
  oncancel={(event) => {
    event.preventDefault();
    close();
  }}
  onclick={(event) => {
    if (event.target === dialog) close();
  }}
  onclose={() => {
    if (open) close();
  }}
>
  <header class={styles.header}>
    <div>
      <h2 class={styles.title}>{title}</h2>
      {#if subtitle}<p class={styles.subtitle}>{subtitle}</p>{/if}
    </div>
    {#if !locked}<Button type="button" variant="ghost" size="sm" class={styles.close} aria-label="Close" onclick={close}>✕</Button>{/if}
  </header>
  <div class={styles.body}>{@render children()}</div>
</dialog>
