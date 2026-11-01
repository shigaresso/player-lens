class InputError extends Error {
  constructor(field, message) {
    super(message);
    this.field = field;
  }
}

// 商品 URL の文字列を解析するだけで、接続先への通信は行わない。
export function parseSteamAppID(value, definition) {
  let url;
  try {
    url = new URL(value.trim());
  } catch {
    throw new InputError('steam_url', 'Steam の商品ページの URL を入力してください。');
  }
  if (url.protocol !== 'https:' || url.hostname !== 'store.steampowered.com'
      || url.username || url.password || url.port) {
    throw new InputError('steam_url', 'https://store.steampowered.com/app/ で始まる Steam 商品 URL を入力してください。');
  }
  const match = url.pathname.match(/^\/app\/([0-9]+)(?:\/.*)?$/);
  const appID = match ? Number(match[1]) : 0;
  if (!Number.isSafeInteger(appID) || appID < definition.minimum || appID > definition.maximum) {
    throw new InputError('steam_url', `App ID は ${definition.minimum}〜${definition.maximum} の整数を指定してください。`);
  }
  return appID;
}
