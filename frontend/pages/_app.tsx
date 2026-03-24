import type { AppProps } from "next/app";
import { useEffect } from "react";

import { applyTheme, readAppearanceSettings } from "@/lib/appearance";

import "@/styles/globals.css";

export default function App({ Component, pageProps }: AppProps) {
  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const syncTheme = () => {
      applyTheme(readAppearanceSettings().theme);
    };

    syncTheme();

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    mediaQuery.addEventListener("change", syncTheme);
    window.addEventListener("storage", syncTheme);

    return () => {
      mediaQuery.removeEventListener("change", syncTheme);
      window.removeEventListener("storage", syncTheme);
    };
  }, []);

  return <Component {...pageProps} />;
}
