<script lang="ts">
import { api } from "../../api";
import { type ErrorPresentation, presentCommandError } from "../../lib/errors";
import { stripHtml } from "../../lib/utils";
import stateStyles from "../../styles/states.module.css";
import type { QueryHit } from "../../types";
import Button from "../../ui/Button.svelte";
import Dialog from "../../ui/Dialog.svelte";
import Field from "../../ui/Field.svelte";
import dialogStyles from "../../ui/dialog.module.css";
import fieldStyles from "../../ui/field.module.css";
import ErrorState from "../ErrorState.svelte";
import styles from "./SearchModal.module.css";

let {
  open = $bindable(),
  onClose,
  onSelectHit,
  onImport,
}: {
  open: boolean;
  onClose: () => void;
  onSelectHit: (page: number, blockId: number) => void;
  onImport?: () => void;
} = $props();
let hits: QueryHit[] = $state([]),
  total: number | null = $state(null),
  loading = $state(false),
  error: ErrorPresentation | null = $state(null),
  searchRequest = 0;

async function submit(event: SubmitEvent) {
  event.preventDefault();
  const params: Record<string, string> = {};
  for (
    const [key, value] of new FormData(event.currentTarget as HTMLFormElement)
  ) {
    if (String(value).trim()) params[key] = String(value);
  }
  const request = ++searchRequest;
  loading = true;
  error = null;
  try {
    const data = await api.search.query(params);
    if (request !== searchRequest) return;
    hits = data.hits;
    total = data.total;
  } catch (cause) {
    if (request !== searchRequest) return;
    error = presentCommandError(cause);
    hits = [];
    total = null;
  } finally {
    if (request === searchRequest) loading = false;
  }
}
</script>

<Dialog bind:open wide title="Search blocks" subtitle="Filter by label, content, or page range"
  onclose={onClose}>
  <form class={styles.form} onsubmit={submit}>
    <div class={fieldStyles.grid}>
      <Field label="Label"
      ><input
          class={fieldStyles.input}
          name="label"
          placeholder="table, text, …"
        /></Field>
      <Field label="Content contains"
      ><input
          class={fieldStyles.input}
          name="content"
          placeholder="keyword"
        /></Field>
      <Field label="Page from"
      ><input
          class={fieldStyles.input}
          name="page_from"
          type="number"
          min="0"
          placeholder="0"
        /></Field>
      <Field label="Page to"
      ><input
          class={fieldStyles.input}
          name="page_to"
          type="number"
          min="0"
          placeholder="821"
        /></Field>
    </div>
    <Button type="submit" variant="primary"
      disabled={loading}>Run query</Button>
  </form>
  <div class={dialogStyles.results}>
    {#if loading}<div class={stateStyles.loading}>Searching…</div>{/if}
    {#if error}<div class={styles.statusError}>
        <ErrorState presentation={error} {onImport} />
      </div>{/if}
    {#if total !== null && !loading}<div>
        <p class="muted">{total} result{total === 1 ? "" : "s"}</p>
        <ul class={styles.list}>
          {#each hits as hit (`${hit.page_index}-${hit.block_id}`)}<li>
              <Button
                type="button"
                variant="bare"
                class={styles.hit}
                onclick={() => {
                  open = false;
                  onClose();
                  onSelectHit(hit.page_index, hit.block_id);
                }}
                ><span class={styles.id}>p{hit.page_index}</span><span class={styles.chip}
                  >{hit.label} · #{hit.block_id}</span
                ><span class={styles.preview}>{stripHtml(hit.preview)}</span></Button
              >
            </li>{/each}
        </ul>
      </div>{/if}
  </div>
</Dialog>
