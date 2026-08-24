import { useEffect, useRef, useState } from "react";
import { client, getApiErrorMessage } from "@/api/client";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import {
  Bot,
  Loader2,
  MessageCircle,
  Send,
  Sparkles,
  User,
  X,
} from "lucide-react";
import type { ChatMessage, PendingAction } from "@/types";
import { cn } from "@/lib/utils";

interface ChatPanelProps {
  onActionComplete?: () => void;
}

const WELCOME: ChatMessage = {
  role: "assistant",
  content:
    "Hola, soy tu asistente de HNL Bank. Pregúntame por tu saldo, movimientos o pídeme hacer un depósito, retiro o transferencia.",
};

function bodyFor(pending: PendingAction): Record<string, unknown> {
  if (pending && typeof pending === "object") return pending;
  return {};
}

export function ChatPanel({ onActionComplete }: ChatPanelProps) {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([WELCOME]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [pending, setPending] = useState<PendingAction>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading, open]);

  const pushMessage = (msg: ChatMessage) => {
    setMessages((prev) => [...prev, msg]);
  };

  const resetConversation = () => {
    setMessages([WELCOME]);
    setPending(null);
  };

  const send = async (raw?: string) => {
    const text = (raw ?? input).trim();
    if (!text || loading) return;
    const apiHistory = messages
      .filter((m) => m.role !== "system")
      .map((m) => ({ role: m.role, content: m.content }));
    if (raw === undefined) setInput("");
    pushMessage({ role: "user", content: text });
    setLoading(true);
    setPending(null);
    try {
      const res = await client.post<{
        message: string;
        requires_confirmation?: boolean;
        pending_action?: PendingAction;
      }>("/chat", {
        message: text,
        history: apiHistory,
      });
      const data = res.data;
      pushMessage({ role: "assistant", content: data.message || "Listo." });
      if (data.requires_confirmation && data.pending_action) {
        setPending(data.pending_action);
      }
    } catch (err) {
      pushMessage({
        role: "assistant",
        content: getApiErrorMessage(err, "No pude procesar tu solicitud."),
      });
    } finally {
      setLoading(false);
      inputRef.current?.focus();
    }
  };

  const confirm = async () => {
    if (!pending) return;
    setLoading(true);
    try {
      const res = await client.post<{ message?: string }>("/chat/action", bodyFor(pending));
      const data = res.data;
      pushMessage({
        role: "assistant",
        content: data.message || "Acción realizada correctamente.",
      });
      setPending(null);
      toast.success(data.message || "Acción realizada correctamente.");
      onActionComplete?.();
    } catch (err) {
      pushMessage({
        role: "assistant",
        content: getApiErrorMessage(err, "No se pudo completar la acción."),
      });
    } finally {
      setLoading(false);
    }
  };

  const cancel = () => {
    setPending(null);
    pushMessage({ role: "assistant", content: "Acción cancelada." });
  };

  return (
    <>
      {/* Floating toggle button */}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="fixed bottom-5 right-5 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105"
        aria-label="Abrir chat con IA"
      >
        {open ? <X className="h-6 w-6" /> : <MessageCircle className="h-6 w-6" />}
      </button>

      {open && (
        <div className="fixed inset-0 z-50 flex justify-end bg-black/40" onClick={() => setOpen(false)}>
          <div
            className="flex h-full w-full max-w-md flex-col bg-background shadow-xl sm:border-l"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div className="flex items-center justify-between border-b px-4 py-3">
              <div className="flex items-center gap-2 font-semibold">
                <Sparkles className="h-5 w-5 text-primary" />
                Asistente IA
              </div>
              <div className="flex items-center gap-1">
                <Button variant="ghost" size="sm" onClick={resetConversation}>
                  Limpiar
                </Button>
                <Button variant="ghost" size="icon" onClick={() => setOpen(false)}>
                  <X className="h-5 w-5" />
                </Button>
              </div>
            </div>

            {/* Messages */}
            <div className="flex-1 space-y-3 overflow-y-auto p-4">
              {messages.map((m, i) => (
                <div
                  key={i}
                  className={cn(
                    "flex w-full gap-2",
                    m.role === "user" ? "justify-end" : "justify-start"
                  )}
                >
                  {m.role === "assistant" && (
                    <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted">
                      <Bot className="h-4 w-4" />
                    </div>
                  )}
                  <div
                    className={cn(
                      "max-w-[75%] rounded-2xl px-3 py-2 text-sm",
                      m.role === "user"
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-foreground"
                    )}
                  >
                    {m.content}
                  </div>
                  {m.role === "user" && (
                    <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10">
                      <User className="h-4 w-4" />
                    </div>
                  )}
                </div>
              ))}

              {loading && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Pensando…
                </div>
              )}

              {/* Confirmation actions */}
              {pending && (
                <div className="rounded-lg border bg-muted/50 p-3">
                  <p className="mb-2 text-sm font-medium">
                    La IA quiere ejecutar una acción. ¿La confirmas?
                  </p>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={confirm} disabled={loading}>
                      Confirmar
                    </Button>
                    <Button size="sm" variant="outline" onClick={cancel} disabled={loading}>
                      Cancelar
                    </Button>
                  </div>
                </div>
              )}

              <div ref={bottomRef} />
            </div>

            {/* Input */}
            <form
              className="flex items-center gap-2 border-t p-3"
              onSubmit={(e) => {
                e.preventDefault();
                send();
              }}
            >
              <input
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Escribe un mensaje…"
                className="h-9 flex-1 rounded-md border border-input bg-transparent px-3 py-1 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
                disabled={loading}
              />
              <Button type="submit" size="icon" disabled={loading || !input.trim()}>
                <Send className="h-4 w-4" />
              </Button>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
