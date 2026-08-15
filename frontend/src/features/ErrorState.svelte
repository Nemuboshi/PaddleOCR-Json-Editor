<script lang="ts">
import type { ErrorPresentation } from "../lib/errors";
import stateStyles from "../styles/states.module.css";
import Button from "../ui/Button.svelte";

let {
  presentation,
  onImport,
  onRetry,
}: {
  presentation: ErrorPresentation;
  onImport?: (() => void) | undefined;
  onRetry?: (() => void) | undefined;
} = $props();
</script>

<div class={`${stateStyles.empty} ${stateStyles.compact}`}>
  <p>{presentation.message}</p>
  {#if presentation.hint}<p class={`muted ${stateStyles.hint}`}>{presentation.hint}</p>{/if}
  {#if presentation.action === "import" && onImport}<Button
      type="button"
      variant="primary"
      size="sm"
      onclick={onImport}>Import JSON</Button
    >{/if}
  {#if presentation.action === "retry" && onRetry}<Button
      type="button"
      variant="primary"
      size="sm"
      onclick={onRetry}>Retry</Button
    >{/if}
</div>
