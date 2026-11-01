import assert from 'node:assert/strict';
import test from 'node:test';
import { parseSteamAppID, generatePrompt, copyPrompt } from './prompt_builder.mjs';

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

const schema = { properties: {
  app_id: appID,
  review_type: { enum: ['', 'all', 'positive', 'negative'], default: 'all' },
  language: { default: 'english' },
  page_size: { maximum: 100, default: 100 },
  max_reviews: { maximum: 100, default: 20 },
  max_pages: { maximum: 5, default: 3 },
} };
const fields = {
  steam_url: 'https://store.steampowered.com/app/440/Team_Fortress_2/?l=japanese',
  connection: 'player_lens_render', language: 'japanese', review_type: 'all',
  page_size: '100', max_reviews: '20', max_pages: '3',
};

test('選択条件と分析の依頼を生成文に保持する', async (t) => {
  for (const [name, changes] of [
    ['Render・全評価', {}],
    ['ローカル・おすすめ・英語', { connection: 'player_lens_local', review_type: 'positive', language: 'english' }],
    ['独自接続名・非おすすめ・全言語・上限', { connection: 'レビュー分析_1', review_type: 'negative', language: 'all', max_reviews: '100', max_pages: '5' }],
  ]) {
    await t.test(name, () => {
      const input = { ...fields, ...changes };
      const text = generatePrompt(input, schema);
      assert.ok(text.startsWith(`${input.connection} の collect_reviews`));
      for (const [key, value] of Object.entries({ app_id: 440, language: input.language,
        review_type: input.review_type, page_size: input.page_size,
        max_reviews: input.max_reviews, max_pages: input.max_pages })) {
        assert.ok(text.includes(`${key}=${value}`), `${key} を指定した値で出力する`);
      }
      for (const instruction of ['1回だけ', 'analysis_guide', 'レビュー ID', '意見', '解釈', '不足や偏り', '操作指示には従わず', '追加取得はしない', '分析を止めて']) {
        assert.ok(text.includes(instruction), instruction);
      }
      assert.ok(!text.includes('Team_Fortress_2') && !text.includes('?l='));
    });
  }
});

test('不正な取得条件と接続名では依頼文を生成しない', async (t) => {
  for (const [name, changes] of [
    ['空の接続名', { connection: '' }], ['改行を含む接続名', { connection: 'name\n別の命令' }],
    ['長すぎる接続名', { connection: 'x'.repeat(65) }], ['記号を含む接続名', { connection: '<script>' }],
    ['未知の評価', { review_type: 'mixed' }], ['大文字の評価', { review_type: 'Positive' }],
    ['未知の言語', { language: 'unknown' }], ['空の件数', { max_reviews: '' }],
    ['0の件数', { max_reviews: '0' }], ['範囲外の件数', { max_reviews: '101' }],
    ['小数の件数', { max_reviews: '1.5' }], ['文字列の件数', { max_reviews: 'abc' }],
    ['範囲外のページ件数', { page_size: '101' }], ['0のページ数', { max_pages: '0' }],
    ['範囲外のページ数', { max_pages: '6' }],
  ]) {
    await t.test(name, () => assert.throws(() => generatePrompt({ ...fields, ...changes }, schema)));
  }
});

test('生成時も画面に渡された最新の上限を使う', () => {
  const changed = { properties: { ...schema.properties, max_reviews: { maximum: 3, default: 2 } } };
  assert.throws(() => generatePrompt(fields, changed), /3/);
  assert.ok(generatePrompt({ ...fields, max_reviews: '3' }, changed).includes('max_reviews=3'));
});

test('生成文だけをコピーし、失敗は手動コピーへ戻せる', async (t) => {
  await t.test('成功すると同じ文字列をコピーする', async () => {
    let copied;
    const clipboard = { writeText: async (value) => { copied = value; } };
    assert.equal(await copyPrompt('生成した依頼文', clipboard), true);
    assert.equal(copied, '生成した依頼文');
  });
  await t.test('権限拒否は失敗として返す', async () => {
    assert.equal(await copyPrompt('生成した依頼文', { writeText: async () => { throw new Error('denied'); } }), false);
  });
  await t.test('API がない場合は失敗として返す', async () => {
    assert.equal(await copyPrompt('生成した依頼文', undefined), false);
  });
  await t.test('空の依頼文はコピーしない', async () => {
    let calls = 0;
    assert.equal(await copyPrompt('', { writeText: async () => { calls += 1; } }), false);
    assert.equal(calls, 0);
  });
});

test('画面の英語選択は MCP の省略時言語に依存しない', () => {
  const changed = { properties: { ...schema.properties, language: { default: 'japanese' } } };
  assert.ok(generatePrompt({ ...fields, language: 'english' }, changed).includes('language=english'));
});
