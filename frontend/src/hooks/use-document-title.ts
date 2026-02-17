import { useCallback, useEffect, useRef, useState } from "react";

const BASE_TITLE = "ChatSphere";

export function useDocumentTitle() {
  const [unreadCount, setUnreadCount] = useState(0);
  const isVisibleRef = useRef(!document.hidden);

  useEffect(() => {
    function handleVisibilityChange() {
      const visible = !document.hidden;
      isVisibleRef.current = visible;
      if (visible) {
        setUnreadCount(0);
        document.title = BASE_TITLE;
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      document.title = BASE_TITLE;
    };
  }, []);

  useEffect(() => {
    if (unreadCount > 0) {
      document.title = `(${unreadCount}) ${BASE_TITLE}`;
    } else {
      document.title = BASE_TITLE;
    }
  }, [unreadCount]);

  const incrementUnread = useCallback(() => {
    if (!isVisibleRef.current) {
      setUnreadCount((prev) => prev + 1);
    }
  }, []);

  return { unreadCount, incrementUnread };
}
