const BANNER = String.raw`
 ____  _____ _____ _____  __  ____ _   _    _  _____
|  _ \| ____| ____|_ _\ \/ / / ___| | | |  / \|_   _|
| | | |  _| |  _|  | | \  / | |   | |_| | / _ \ | |
| |_| | |___| |___ | | /  \ | |___|  _  |/ ___ \| |
|____/|_____|_____|___/_/\_\ \____|_| |_/_/   \_\_|
`;

const BANNER_SCRIPT = `
(() => {
  const key = "__DEEIX_CHAT_DEVTOOLS_BANNER__";
  if (globalThis[key]) return;
  globalThis[key] = true;
  const banner = ${JSON.stringify(BANNER)};
  const mono = "font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace";
  console.log("%c" + banner, mono + ";font-weight:700;line-height:1.15");
  console.log("%cOfficial: https://deeix.com  |  Repository: https://github.com/DEEIX-AI/DEEIX-Chat  |  License: Apache License 2.0", mono);
})();
`;

export function DevtoolsBrandBanner() {
  return (
    <script
      id="deeix-devtools-brand"
      dangerouslySetInnerHTML={{ __html: BANNER_SCRIPT }}
    />
  );
}
