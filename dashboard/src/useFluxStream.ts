import { useEffect, useRef, useState } from "react";
import type { AIDecision, DigestResult } from "./api";

export type StreamMessage =
  | { type: "event"; data: unknown }
  | { type: "digest"; data: DigestResult }
  | { type: "decision"; data: AIDecision }
  | { type: "alert"; data: unknown };

function wsURL(): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/api/v1/stream`;
}

export function useFluxStream(enabled: boolean) {
  const [liveDigest, setLiveDigest] = useState<DigestResult | null>(null);
  const [decisions, setDecisions] = useState<AIDecision[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!enabled) return;
    const ws = new WebSocket(wsURL());
    wsRef.current = ws;
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string) as { type: string; data: unknown };
        if (msg.type === "digest") {
          setLiveDigest(msg.data as DigestResult);
        } else if (msg.type === "decision") {
          const d = msg.data as AIDecision;
          setDecisions((prev) => {
            const next = [...prev, d];
            return next.length > 200 ? next.slice(-200) : next;
          });
        }
      } catch {
        /* ignore malformed */
      }
    };
    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [enabled]);

  return { liveDigest, decisions, connected };
}
