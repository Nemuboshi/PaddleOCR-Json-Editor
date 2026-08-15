<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLButtonAttributes } from "svelte/elements";
import styles from "./button.module.css";

let {
  variant = "default",
  size = "md",
  class: className = "",
  children,
  ...props
}: {
  variant?: "default" | "primary" | "ghost" | "danger" | "bare";
  size?: "md" | "sm";
  class?: string | undefined;
  children: Snippet;
} & HTMLButtonAttributes = $props();

const classes = $derived(
  [
    styles.btn,
    variant === "primary" && styles.primary,
    variant === "ghost" && styles.ghost,
    variant === "danger" && styles.danger,
    variant === "bare" && styles.bare,
    size === "sm" && styles.sm,
    className,
  ]
    .filter(Boolean)
    .join(" "),
);
</script>

<button class={classes} {...props}>{@render children()}</button>
