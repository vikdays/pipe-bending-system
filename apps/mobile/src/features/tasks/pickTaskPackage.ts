import * as DocumentPicker from 'expo-document-picker';
import { File, Paths } from 'expo-file-system';
import { MAX_PACKAGE_BYTES, parseTaskPackage } from './taskPackage';

export async function pickTaskPackage() {
  const result = await DocumentPicker.getDocumentAsync({ type: '*/*', multiple: false, copyToCacheDirectory: true });
  if (result.canceled) return null;
  const asset = result.assets[0];
  const file = new File(asset.uri);
  try {
    if (!asset.name.toLowerCase().endsWith('.json')) throw new Error('Выберите JSON-пакет заданий. Файл SQLite, PDF или ZIP для импорта не подходит.');
    if ((asset.size ?? file.size) > MAX_PACKAGE_BYTES) throw new Error('Максимальный размер пакета — 8 МБ.');
    const text = await file.text();
    return { text, name: asset.name, data: parseTaskPackage(text) };
  } finally {
    // Only remove the picker-created cache copy, never the original document.
    if (file.uri.startsWith(`${Paths.cache.uri.replace(/\/$/, '')}/`)) {
      try { await file.delete(); } catch { /* Cleanup must not change the import result. */ }
    }
  }
}
export type PackagePreview = NonNullable<Awaited<ReturnType<typeof pickTaskPackage>>>;
