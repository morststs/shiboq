// Web版の保存項目（デスクトップ版のsaved_service.goに相当）。IndexedDBに保存する。
// localStorageは容量が5MB程度しかなく、JSONを丸ごと保存すると溢れやすいため使わない。
// 振る舞いはsaved_service.goに揃える: IDが空ならUUIDを発行し、内容が同じでも
// 必ず新規保存する。一覧は新しい順（同時刻はID降順）で、UUIDでない項目は除外する。
const DB_NAME = 'shiboq';
const STORE = 'saved';
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

let dbPromise = null;
function openDb() {
  if (!dbPromise) {
    dbPromise = new Promise((resolve, reject) => {
      const req = indexedDB.open(DB_NAME, 1);
      req.onupgradeneeded = () => {
        req.result.createObjectStore(STORE, { keyPath: 'id' });
      };
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
    dbPromise.catch(() => {
      dbPromise = null;
    });
  }
  return dbPromise;
}

async function run(mode, fn) {
  const db = await openDb();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, mode);
    const req = fn(tx.objectStore(STORE));
    let value;
    req.onsuccess = () => {
      value = req.result;
    };
    // 書き込みはトランザクション完了まで待ってから成功とみなす
    tx.oncomplete = () => resolve(value);
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error);
  });
}

export async function SaveItem(item) {
  let id = (item.id ?? '').trim();
  if (id === '') id = crypto.randomUUID();
  else if (!UUID_RE.test(id)) throw new Error(`invalid saved item ID: ${id}`);
  const saved = {
    id,
    name: item.name ?? '',
    json: item.json ?? '',
    query: item.query ?? '',
    result: item.result ?? '',
    createdAt: item.createdAt || new Date().toISOString(),
  };
  await run('readwrite', (s) => s.put(saved));
  return saved;
}

export async function ListSaved() {
  const all = (await run('readonly', (s) => s.getAll())) ?? [];
  const items = all.filter((i) => typeof i.id === 'string' && UUID_RE.test(i.id));
  items.sort((a, b) => {
    const ta = Date.parse(a.createdAt);
    const tb = Date.parse(b.createdAt);
    if (ta !== tb) return tb - ta;
    return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
  });
  return items;
}

export async function DeleteSaved(id) {
  if (!UUID_RE.test(id ?? '')) throw new Error(`invalid saved item ID: ${id}`);
  const existing = await run('readonly', (s) => s.get(id));
  if (!existing) throw new Error(`saved item not found: ${id}`);
  await run('readwrite', (s) => s.delete(id));
}
