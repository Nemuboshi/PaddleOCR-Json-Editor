<script lang="ts">
import type { Snippet } from "svelte";
import styles from "./field.module.css";

let {
  label,
  class: className = "",
  children,
}: { label: string; class?: string; children: Snippet } = $props();
</script>

<label class={`${styles.field} ${className}`}>
  <span class={styles.label}>{label}</span>
  {@render children()}
</label>
