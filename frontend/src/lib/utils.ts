export const LABEL_COLORS: Record<string, string> = {
  text: "#4f8cff",
  paragraph_title: "#ffb020",
  doc_title: "#ff6b6b",
  table: "#51cf66",
  image: "#cc5de8",
  header: "#868e96",
  footer: "#868e96",
  header_image: "#cc5de8",
  footer_image: "#cc5de8",
  display_formula: "#20c997",
  vision_footnote: "#fab005",
};

export function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

export function stripHtml(value: string): string {
  const element = document.createElement("div");
  element.innerHTML = value;
  return element.innerText.replace(/\s+/g, " ").trim();
}

export function compactFileName(path: string, maxLength = 40): string {
  const name = path.split(/[\\/]/).at(-1) || path;
  const characters = Array.from(name);
  if (characters.length <= maxLength) return name;
  const start = Math.ceil((maxLength - 1) / 2);
  return `${characters.slice(0, start).join("")}…${characters.slice(start + 1 - maxLength).join("")}`;
}
