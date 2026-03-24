import { Html, Head, Main, NextScript } from "next/document";

import { APPEARANCE_STORAGE_KEY } from "@/lib/appearance";

const themeBootstrapScript = `
  (function () {
    try {
      var theme = "system";
      var rawValue = window.localStorage.getItem("${APPEARANCE_STORAGE_KEY}");
      if (rawValue) {
        var parsed = JSON.parse(rawValue);
        if (parsed && typeof parsed.theme === "string") {
          theme = parsed.theme;
        }
      }

      var resolvedTheme = theme;
      if (theme === "system") {
        resolvedTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
      }

      var root = document.documentElement;
      root.classList.toggle("dark", resolvedTheme === "dark");
      root.classList.toggle("light", resolvedTheme === "light");
    } catch (error) {
      document.documentElement.classList.add("dark");
    }
  })();
`;

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </Head>
      <body>
        <script dangerouslySetInnerHTML={{ __html: themeBootstrapScript }} />
        <Main />
        <NextScript />
      </body>
    </Html>
  );
}
