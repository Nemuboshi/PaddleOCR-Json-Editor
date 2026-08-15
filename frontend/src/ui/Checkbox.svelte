<script lang="ts">
import type { Snippet } from "svelte";
import styles from "./checkbox.module.css";

let {
  id,
  name,
  checked = $bindable(false),
  label,
  class: className = "",
}: {
  id?: string | undefined;
  name?: string | undefined;
  checked?: boolean;
  label: string | Snippet;
  class?: string | undefined;
} = $props();
</script>

<label class={`${styles.root} ${className}`}>
  <input class={styles.box} type="checkbox" {id} {name} bind:checked />
  <span>{#if typeof label === "string"}{label}{:else}{@render label()}{/if}</span>
</label>
