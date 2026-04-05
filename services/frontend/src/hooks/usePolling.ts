import { useEffect, useRef } from 'react';

/**
 * Polls a callback at `intervalMs` (default 30 s) while the component is mounted.
 * Stops when the component unmounts.
 */
export function usePolling(callback: () => void, intervalMs = 30_000) {
  const savedCb = useRef(callback);
  useEffect(() => {
    savedCb.current = callback;
  }, [callback]);

  useEffect(() => {
    const id = setInterval(() => savedCb.current(), intervalMs);
    return () => clearInterval(id);
  }, [intervalMs]);
}
