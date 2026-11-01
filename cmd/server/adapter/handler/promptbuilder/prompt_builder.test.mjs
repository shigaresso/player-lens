import assert from 'node:assert/strict';
import test from 'node:test';
import { parseSteamAppID } from './prompt_builder.mjs';

const appID = { minimum: 1, maximum: 4294967295 };

test('Steam 商品 URL から App ID を取り出す', async (t) => {
  for (const [name, url, expected] of [
    ['数字のみ', 'https://store.steampowered.com/app/440', 440],
    ['作品名・query・fragmentを無視する', 'https://store.steampowered.com/app/440/Team_Fortress_2/?l=japanese#reviews', 440],
    ['前後空白と先頭ゼロを正規化する', '  https://store.steampowered.com/app/000440/  ', 440],
    ['最小値', 'https://store.steampowered.com/app/1/', 1],
    ['最大値', 'https://store.steampowered.com/app/4294967295/', 4294967295],
  ]) {
    await t.test(name, () => assert.equal(parseSteamAppID(url, appID), expected));
  }
});

test('対応しない URL と不正な App ID を拒否する', async (t) => {
  for (const [name, url] of [
    ['空', ''], ['相対パス', '/app/440/'],
    ['HTTP', 'http://store.steampowered.com/app/440/'],
    ['別ホスト', 'https://example.com/app/440/'],
    ['紛らわしいホスト', 'https://store.steampowered.com.example.com/app/440/'],
    ['認証情報', 'https://user:pass@store.steampowered.com/app/440/'],
    ['別ポート', 'https://store.steampowered.com:444/app/440/'],
    ['商品以外', 'https://store.steampowered.com/bundle/440/'],
    ['0', 'https://store.steampowered.com/app/0/'],
    ['範囲外', 'https://store.steampowered.com/app/4294967296/'],
    ['小数', 'https://store.steampowered.com/app/44.0/'],
    ['数字の後ろに文字', 'https://store.steampowered.com/app/440evil/'],
  ]) {
    await t.test(name, () => assert.throws(() => parseSteamAppID(url, appID), /Steam|App ID/));
  }
});

test('App ID の範囲には渡された定義を使う', () => {
  assert.throws(() => parseSteamAppID('https://store.steampowered.com/app/440/', { minimum: 1, maximum: 100 }), /App ID/);
});
