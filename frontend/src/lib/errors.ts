import { type ErrorAction, errorText } from "../copy";

type CommandErrorBody = { code: string; message: string };

export interface ErrorPresentation {
  code: string | null;
  message: string;
  hint: string | null;
  action: ErrorAction;
}

function parseCommandError(err: unknown): CommandErrorBody | null {
  const text =
    typeof err === "string" ? err : err instanceof Error ? err.message : null;
  if (text) {
    const json = text.slice(text.indexOf("{"));
    try {
      const parsed = JSON.parse(json) as CommandErrorBody;
      if (parsed.code && parsed.message) return parsed;
    } catch {
      return null;
    }
  }
  if (err && typeof err === "object") {
    const body = err as Partial<CommandErrorBody>;
    if (typeof body.code === "string" && typeof body.message === "string") {
      return { code: body.code, message: body.message };
    }
  }
  return null;
}

export function presentCommandError(err: unknown): ErrorPresentation {
  const parsed = parseCommandError(err);
  const message =
    parsed?.message ?? (err instanceof Error ? err.message : String(err));
  const code = parsed?.code ?? null;
  const mapped = code ? errorText[code] : undefined;

  return {
    code,
    message,
    hint: mapped?.hint ?? null,
    action: mapped?.action ?? null,
  };
}
