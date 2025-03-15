import { useEffect, useRef } from 'react';

/**
 * Custom hook for setting up an interval that is properly cleaned up when component unmounts
 * @param {Function} callback - The function to call on each interval
 * @param {number} delay - The delay in milliseconds (null to pause)
 */
export function useInterval(callback, delay) {
  const savedCallback = useRef();

  // Remember the latest callback
  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  // Set up the interval
  useEffect(() => {
    function tick() {
      savedCallback.current();
    }
    if (delay !== null) {
      const id = setInterval(tick, delay);
      return () => clearInterval(id);
    }
  }, [delay]);
}
